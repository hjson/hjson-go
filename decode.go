package hjson

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const maxPointerDepth = 512

// If a destination type implements ElemTyper, Unmarshal() will call ElemType()
// on the destination when unmarshalling an array or an object, to see if any
// array element or leaf node should be of type string even if it can be treated
// as a number, boolean or null. This is most useful if the destination also
// implements the json.Unmarshaler interface, because then there is no other way
// for Unmarshal() to know the type of the elements on the destination. If a
// destination implements ElemTyper all of its elements must be of the same
// type.
type ElemTyper interface {
	// Returns the desired type of any elements. If ElemType() is implemented
	// using a pointer receiver it must be possible to call with nil as receiver.
	ElemType() reflect.Type
}

// DecoderOptions defines options for decoding Hjson.
type DecoderOptions struct {
	// UseJSONNumber causes the Decoder to unmarshal a number into an interface{} as a
	// json.Number instead of as a float64.
	UseJSONNumber bool
	// DisallowUnknownFields causes an error to be returned when the destination
	// is a struct and the input contains object keys which do not match any
	// non-ignored, exported fields in the destination.
	DisallowUnknownFields bool
}

// DefaultDecoderOptions returns the default decoding options.
func DefaultDecoderOptions() DecoderOptions {
	return DecoderOptions{
		UseJSONNumber:         false,
		DisallowUnknownFields: false,
	}
}

type hjsonParser struct {
	DecoderOptions
	data              []byte
	at                int  // The index of the current character
	ch                byte // The current character
	structTypeCache   map[reflect.Type]structFieldMap
	willMarshalToJSON bool
}

var unmarshalerText = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
var elemTyper = reflect.TypeOf((*ElemTyper)(nil)).Elem()

func (p *hjsonParser) resetAt() {
	p.at = 0
	p.ch = ' '
}

func isPunctuatorChar(c byte) bool {
	return c == '{' || c == '}' || c == '[' || c == ']' || c == ',' || c == ':'
}

func (p *hjsonParser) errAt(message string) error {
	var i int
	col := 0
	line := 1
	for i = p.at - 1; i > 0 && p.data[i] != '\n'; i-- {
		col++
	}
	for ; i > 0; i-- {
		if p.data[i] == '\n' {
			line++
		}
	}
	samEnd := p.at - col + 20
	if samEnd > len(p.data) {
		samEnd = len(p.data)
	}
	return fmt.Errorf("%s at line %d,%d >>> %s", message, line, col, string(p.data[p.at-col:samEnd]))
}

func (p *hjsonParser) next() bool {
	// get the next character.
	if p.at < len(p.data) {
		p.ch = p.data[p.at]
		p.at++
		return true
	}
	p.ch = 0
	return false
}

func (p *hjsonParser) peek(offs int) byte {
	pos := p.at + offs
	if pos >= 0 && pos < len(p.data) {
		return p.data[p.at+offs]
	}
	return 0
}

var escapee = map[byte]byte{
	'"':  '"',
	'\'': '\'',
	'\\': '\\',
	'/':  '/',
	'b':  '\b',
	'f':  '\f',
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
}

func unravelDestination(dest reflect.Value, t reflect.Type) (reflect.Value, reflect.Type) {
	if dest.IsValid() {
		for a := 0; a < maxPointerDepth && (dest.Kind() == reflect.Ptr ||
			dest.Kind() == reflect.Interface) && !dest.IsNil(); a++ {

			dest = dest.Elem()
		}

		if dest.IsValid() {
			t = dest.Type()
		}
	}

	for a := 0; a < maxPointerDepth && t != nil && t.Kind() == reflect.Ptr; a++ {
		t = t.Elem()
	}

	return dest, t
}

func (p *hjsonParser) readString(allowML bool) (string, error) {

	// Parse a string value.
	res := new(bytes.Buffer)

	// callers make sure that (ch === '"' || ch === "'")
	// When parsing for string values, we must look for " and \ characters.
	exitCh := p.ch
	for p.next() {
		if p.ch == exitCh {
			p.next()
			if allowML && exitCh == '\'' && p.ch == '\'' && res.Len() == 0 {
				// ''' indicates a multiline string
				p.next()
				return p.readMLString()
			} else {
				return res.String(), nil
			}
		}
		if p.ch == '\\' {
			p.next()
			if p.ch == 'u' {
				uffff := 0
				for i := 0; i < 4; i++ {
					p.next()
					var hex int
					if p.ch >= '0' && p.ch <= '9' {
						hex = int(p.ch - '0')
					} else if p.ch >= 'a' && p.ch <= 'f' {
						hex = int(p.ch - 'a' + 0xa)
					} else if p.ch >= 'A' && p.ch <= 'F' {
						hex = int(p.ch - 'A' + 0xa)
					} else {
						return "", p.errAt("Bad \\u char " + string(p.ch))
					}
					uffff = uffff*16 + hex
				}
				res.WriteRune(rune(uffff))
			} else if ech, ok := escapee[p.ch]; ok {
				res.WriteByte(ech)
			} else {
				return "", p.errAt("Bad escape \\" + string(p.ch))
			}
		} else if p.ch == '\n' || p.ch == '\r' {
			return "", p.errAt("Bad string containing newline")
		} else {
			res.WriteByte(p.ch)
		}
	}
	return "", p.errAt("Bad string")
}

func (p *hjsonParser) readMLString() (value string, err error) {

	// Parse a multiline string value.
	res := new(bytes.Buffer)
	triple := 0

	// we are at ''' +1 - get indent
	indent := 0
	for {
		c := p.peek(-indent - 5)
		if c == 0 || c == '\n' {
			break
		}
		indent++
	}

	skipIndent := func() {
		skip := indent
		for p.ch > 0 && p.ch <= ' ' && p.ch != '\n' && skip > 0 {
			skip--
			p.next()
		}
	}

	// skip white/to (newline)
	for p.ch > 0 && p.ch <= ' ' && p.ch != '\n' {
		p.next()
	}
	if p.ch == '\n' {
		p.next()
		skipIndent()
	}

	// When parsing multiline string values, we must look for ' characters.
	lastLf := false
	for {
		if p.ch == 0 {
			return "", p.errAt("Bad multiline string")
		} else if p.ch == '\'' {
			triple++
			p.next()
			if triple == 3 {
				sres := res.Bytes()
				if lastLf {
					return string(sres[0 : len(sres)-1]), nil // remove last EOL
				}
				return string(sres), nil
			}
			continue
		} else {
			for triple > 0 {
				res.WriteByte('\'')
				triple--
				lastLf = false
			}
		}
		if p.ch == '\n' {
			res.WriteByte('\n')
			lastLf = true
			p.next()
			skipIndent()
		} else {
			if p.ch != '\r' {
				res.WriteByte(p.ch)
				lastLf = false
			}
			p.next()
		}
	}
}

func (p *hjsonParser) readKeyname() (string, error) {

	// quotes for keys are optional in Hjson
	// unless they include {}[],: or whitespace.

	if p.ch == '"' || p.ch == '\'' {
		return p.readString(false)
	}

	name := new(bytes.Buffer)
	start := p.at
	space := -1
	for {
		if p.ch == ':' {
			if name.Len() == 0 {
				return "", p.errAt("Found ':' but no key name (for an empty key name use quotes)")
			} else if space >= 0 && space != name.Len() {
				p.at = start + space
				return "", p.errAt("Found whitespace in your key name (use quotes to include)")
			}
			return name.String(), nil
		} else if p.ch <= ' ' {
			if p.ch == 0 {
				return "", p.errAt("Found EOF while looking for a key name (check your syntax)")
			}
			if space < 0 {
				space = name.Len()
			}
		} else {
			if isPunctuatorChar(p.ch) {
				return "", p.errAt("Found '" + string(p.ch) + "' where a key name was expected (check your syntax or use quotes if the key name includes {}[],: or whitespace)")
			}
			name.WriteByte(p.ch)
		}
		p.next()
	}
}

func (p *hjsonParser) white() {
	for p.ch > 0 {
		// Skip whitespace.
		for p.ch > 0 && p.ch <= ' ' {
			p.next()
		}
		// Hjson allows comments
		if p.ch == '#' || p.ch == '/' && p.peek(0) == '/' {
			for p.ch > 0 && p.ch != '\n' {
				p.next()
			}
		} else if p.ch == '/' && p.peek(0) == '*' {
			p.next()
			p.next()
			for p.ch > 0 && !(p.ch == '*' && p.peek(0) == '/') {
				p.next()
			}
			if p.ch > 0 {
				p.next()
				p.next()
			}
		} else {
			break
		}
	}
}

func (p *hjsonParser) readTfnns(dest reflect.Value, t reflect.Type) (interface{}, error) {

	// Hjson strings can be quoteless
	// returns string, (json.Number or float64), true, false, or null.

	if isPunctuatorChar(p.ch) {
		return nil, p.errAt("Found a punctuator character '" + string(p.ch) + "' when expecting a quoteless string (check your syntax)")
	}
	chf := p.ch
	value := new(bytes.Buffer)
	value.WriteByte(p.ch)

	// Keep the original dest and t, because we need to check if it implements
	// encoding.TextUnmarshaler.
	_, newT := unravelDestination(dest, t)

	for {
		p.next()
		isEol := p.ch == '\r' || p.ch == '\n' || p.ch == 0
		if isEol ||
			p.ch == ',' || p.ch == '}' || p.ch == ']' ||
			p.ch == '#' ||
			p.ch == '/' && (p.peek(0) == '/' || p.peek(0) == '*') {

			// Do not output anything else than a string if our destination is a string.
			// Pointer methods can be called if the destination is addressable,
			// therefore we also check if dest.Addr() implements encoding.TextUnmarshaler.
			if (newT == nil || newT.Kind() != reflect.String) &&
				(t == nil || !(t.Implements(unmarshalerText) ||
					dest.CanAddr() && dest.Addr().Type().Implements(unmarshalerText))) {

				switch chf {
				case 'f':
					if strings.TrimSpace(value.String()) == "false" {
						return false, nil
					}
				case 'n':
					if strings.TrimSpace(value.String()) == "null" {
						return nil, nil
					}
				case 't':
					if strings.TrimSpace(value.String()) == "true" {
						return true, nil
					}
				default:
					if chf == '-' || chf >= '0' && chf <= '9' {
						// Always use json.Number if we will marshal to JSON.
						if n, err := tryParseNumber(
							value.Bytes(),
							false,
							p.willMarshalToJSON || p.DecoderOptions.UseJSONNumber,
						); err == nil {
							return n, nil
						}
					}
				}
			}

			if isEol {
				// remove any whitespace at the end (ignored in quoteless strings)
				return strings.TrimSpace(value.String()), nil
			}
		}
		value.WriteByte(p.ch)
	}
}

// t must not have been unraveled
func getElemTyperType(rv reflect.Value, t reflect.Type) reflect.Type {
	var elemType reflect.Type
	isElemTyper := false

	if t != nil && t.Implements(elemTyper) {
		isElemTyper = true
		if t.Kind() == reflect.Ptr {
			// If ElemType() has a value receiver we would get a panic if we call it
			// on a nil pointer.
			if !rv.IsValid() || rv.IsNil() {
				rv = reflect.New(t.Elem())
			}
		} else if !rv.IsValid() {
			rv = reflect.Zero(t)
		}
	}
	if !isElemTyper && rv.CanAddr() {
		rv = rv.Addr()
		if rv.Type().Implements(elemTyper) {
			isElemTyper = true
		}
	}
	if !isElemTyper && t != nil {
		pt := reflect.PtrTo(t)
		if pt.Implements(elemTyper) {
			isElemTyper = true
			rv = reflect.Zero(pt)
		}
	}
	if isElemTyper {
		elemType = rv.Interface().(ElemTyper).ElemType()
	}

	return elemType
}

func (p *hjsonParser) readArray(dest reflect.Value, t reflect.Type) (value interface{}, err error) {

	// Parse an array value.
	// assuming ch == '['

	array := make([]interface{}, 0, 1)

	p.next()
	p.white()

	if p.ch == ']' {
		p.next()
		return array, nil // empty array
	}

	elemType := getElemTyperType(dest, t)

	dest, t = unravelDestination(dest, t)

	// All elements in any existing slice/array will be removed, so we only care
	// about the type of the new elements that will be created.
	if elemType == nil && t != nil && (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) {
		elemType = t.Elem()
	}

	for p.ch > 0 {
		var val interface{}
		if val, err = p.readValue(reflect.Value{}, elemType); err != nil {
			return nil, err
		}
		array = append(array, val)
		p.white()
		// in Hjson the comma is optional and trailing commas are allowed
		if p.ch == ',' {
			p.next()
			p.white()
		}
		if p.ch == ']' {
			p.next()
			return array, nil
		}
		p.white()
	}

	return nil, p.errAt("End of input while parsing an array (did you forget a closing ']'?)")
}

func (p *hjsonParser) readObject(
	withoutBraces bool,
	dest reflect.Value,
	t reflect.Type,
) (value interface{}, err error) {
	// Parse an object value.

	object := NewOrderedMap()

	if !withoutBraces {
		// assuming ch == '{'
		p.next()
	}

	p.white()
	if p.ch == '}' && !withoutBraces {
		p.next()
		return object, nil // empty object
	}

	var stm structFieldMap
	elemType := getElemTyperType(dest, t)

	dest, t = unravelDestination(dest, t)

	if elemType == nil && t != nil {
		switch t.Kind() {
		case reflect.Struct:
			var ok bool
			stm, ok = p.structTypeCache[t]
			if !ok {
				stm = getStructFieldInfoMap(t)
				p.structTypeCache[t] = stm
			}

		case reflect.Map:
			// For any key that we find in our loop here below, the new value fully
			// replaces any old value. So no need for us to dig down into a tree.
			// (This is because we are decoding into a map. If we were decoding into
			// a struct we would need to dig down into a tree, to match the behavior
			// of Golang's JSON decoder.)
			elemType = t.Elem()
		}
	}

	for p.ch > 0 {
		var key string
		if key, err = p.readKeyname(); err != nil {
			return nil, err
		}
		p.white()
		if p.ch != ':' {
			return nil, p.errAt("Expected ':' instead of '" + string(p.ch) + "'")
		}
		p.next()

		var newDest reflect.Value
		var newDestType reflect.Type
		if stm != nil {
			sfi, ok := stm.getField(key)
			if ok {
				// The field might be found on the root struct or in embedded structs.
				newDest, newDestType = dest, t
				for _, i := range sfi.indexPath {
					newDest, newDestType = unravelDestination(newDest, newDestType)

					if newDestType == nil {
						return nil, p.errAt("Internal error")
					}
					newDestType = newDestType.Field(i).Type
					elemType = newDestType

					if newDest.IsValid() {
						if newDest.Kind() != reflect.Struct {
							// We are only keeping track of newDest in case it contains a
							// tree that we will partially update. But here we have not found
							// any tree, so we can ignore newDest and just look at
							// newDestType instead.
							newDest = reflect.Value{}
						} else {
							newDest = newDest.Field(i)
						}
					}
				}
			}
		}

		// duplicate keys overwrite the previous value
		var val interface{}
		if val, err = p.readValue(newDest, elemType); err != nil {
			return nil, err
		}
		object.Append(key, val)
		p.white()
		// in Hjson the comma is optional and trailing commas are allowed
		if p.ch == ',' {
			p.next()
			p.white()
		}
		if p.ch == '}' && !withoutBraces {
			p.next()
			return object, nil
		}
		p.white()
	}

	if withoutBraces {
		return object, nil
	}
	return nil, p.errAt("End of input while parsing an object (did you forget a closing '}'?)")
}

// dest and t must not have been unraveled yet here. In readTfnns we need
// to check if the original type (or a pointer to it) implements
// encoding.TextUnmarshaler.
func (p *hjsonParser) readValue(dest reflect.Value, t reflect.Type) (interface{}, error) {

	// Parse a Hjson value. It could be an object, an array, a string, a number or a word.

	p.white()
	switch p.ch {
	case '{':
		return p.readObject(false, dest, t)
	case '[':
		return p.readArray(dest, t)
	case '"', '\'':
		return p.readString(true)
	default:
		return p.readTfnns(dest, t)
	}
}

func (p *hjsonParser) rootValue(dest reflect.Value) (interface{}, error) {
	// Braces for the root object are optional

	// We have checked that dest is a pointer before calling rootValue().
	// Dereference here because readObject() etc will pass on child destinations
	// without creating pointers.
	dest = dest.Elem()
	t := dest.Type()

	p.white()
	switch p.ch {
	case '{':
		return p.checkTrailing(p.readObject(false, dest, t))
	case '[':
		return p.checkTrailing(p.readArray(dest, t))
	}

	// assume we have a root object without braces
	res, err := p.checkTrailing(p.readObject(true, dest, t))
	if err == nil {
		return res, nil
	}

	// test if we are dealing with a single JSON value instead (true/false/null/num/"")
	p.resetAt()
	if res2, err2 := p.checkTrailing(p.readValue(dest, t)); err2 == nil {
		return res2, nil
	}
	return res, err
}

func (p *hjsonParser) checkTrailing(v interface{}, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}
	p.white()
	if p.ch > 0 {
		return nil, p.errAt("Syntax error, found trailing characters")
	}
	return v, nil
}

// Unmarshal parses the Hjson-encoded data using default options and stores the
// result in the value pointed to by v.
//
// See UnmarshalWithOptions.
func Unmarshal(data []byte, v interface{}) error {
	return UnmarshalWithOptions(data, v, DefaultDecoderOptions())
}

func orderedUnmarshal(
	data []byte,
	v interface{},
	options DecoderOptions,
	willMarshalToJSON bool,
) (
	interface{},
	error,
) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil, fmt.Errorf("Cannot unmarshal into non-pointer %v", reflect.TypeOf(v))
	}

	parser := &hjsonParser{
		DecoderOptions:    options,
		data:              data,
		at:                0,
		ch:                ' ',
		structTypeCache:   map[reflect.Type]structFieldMap{},
		willMarshalToJSON: willMarshalToJSON,
	}
	parser.resetAt()
	value, err := parser.rootValue(rv)
	if err != nil {
		return nil, err
	}

	return value, nil
}

// UnmarshalWithOptions parses the Hjson-encoded data and stores the result
// in the value pointed to by v.
//
// Unless v is of type *hjson.OrderedMap, the Hjson input is internally
// converted to JSON, which is then used as input to the function
// json.Unmarshal().
//
// For more details about the output from this function, see the documentation
// for json.Unmarshal().
func UnmarshalWithOptions(data []byte, v interface{}, options DecoderOptions) error {
	inOM, destinationIsOrderedMap := v.(*OrderedMap)
	if !destinationIsOrderedMap {
		pInOM, ok := v.(**OrderedMap)
		if ok {
			destinationIsOrderedMap = true
			inOM = &OrderedMap{}
			*pInOM = inOM
		}
	}

	value, err := orderedUnmarshal(data, v, options, !destinationIsOrderedMap)
	if err != nil {
		return err
	}

	if destinationIsOrderedMap {
		if outOM, ok := value.(*OrderedMap); ok {
			*inOM = *outOM
			return nil
		}
		return fmt.Errorf("Cannot unmarshal into hjson.OrderedMap: Try %v as destination instead",
			reflect.TypeOf(v))
	}

	// Convert to JSON so we can let json.Unmarshal() handle all destination
	// types (including interfaces json.Unmarshaler and encoding.TextUnmarshaler)
	// and merging.
	buf, err := json.Marshal(value)
	if err != nil {
		return errors.New("Internal error")
	}

	dec := json.NewDecoder(bytes.NewBuffer(buf))
	if options.UseJSONNumber {
		dec.UseNumber()
	}
	if options.DisallowUnknownFields {
		dec.DisallowUnknownFields()
	}

	err = dec.Decode(v)
	if err != nil {
		return err
	}

	return err
}
