package hjson

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// EncoderOptions defines options for encoding to Hjson.
type EncoderOptions struct {
	// End of line, should be either \n or \r\n
	Eol string
	// Place braces on the same line
	BracesSameLine bool
	// Emit braces for the root object
	EmitRootBraces bool
	// Always place string in quotes
	QuoteAlways bool
	// Place string in quotes if it could otherwise be a number, boolean or null
	QuoteAmbiguousStrings bool
	// Indent string
	IndentBy string
	// Base indentation string
	BaseIndentation string
}

// DefaultOptions returns the default encoding options.
func DefaultOptions() EncoderOptions {
	return EncoderOptions{
		Eol:                   "\n",
		BracesSameLine:        false,
		EmitRootBraces:        true,
		QuoteAlways:           false,
		QuoteAmbiguousStrings: true,
		IndentBy:              "  ",
		BaseIndentation:       "",
	}
}

// Start looking for circular references below this depth.
const depthLimit = 1024

type hjsonEncoder struct {
	bytes.Buffer // output
	EncoderOptions
	indent          int
	pDepth          uint
	parents         map[uintptr]struct{} // Starts to be filled after pDepth has reached depthLimit
	structTypeCache map[reflect.Type][]structFieldInfo
}

var JSONNumberType = reflect.TypeOf(json.Number(""))

var needsEscape, needsQuotes, needsEscapeML, startsWithKeyword, needsEscapeName *regexp.Regexp

func init() {
	var commonRange = `\x7f-\x9f\x{00ad}\x{0600}-\x{0604}\x{070f}\x{17b4}\x{17b5}\x{200c}-\x{200f}\x{2028}-\x{202f}\x{2060}-\x{206f}\x{feff}\x{fff0}-\x{ffff}`
	// needsEscape tests if the string can be written without escapes
	needsEscape = regexp.MustCompile(`[\\\"\x00-\x1f` + commonRange + `]`)
	// needsQuotes tests if the string can be written as a quoteless string (includes needsEscape but without \\ and \")
	needsQuotes = regexp.MustCompile(`^\s|^"|^'|^#|^/\*|^//|^\{|^\}|^\[|^\]|^:|^,|\s$|[\x00-\x1f\x7f-\x9f\x{00ad}\x{0600}-\x{0604}\x{070f}\x{17b4}\x{17b5}\x{200c}-\x{200f}\x{2028}-\x{202f}\x{2060}-\x{206f}\x{feff}\x{fff0}-\x{ffff}]`)
	// needsEscapeML tests if the string can be written as a multiline string (like needsEscape but without \n, \\, \", \t)
	needsEscapeML = regexp.MustCompile(`'''|^[\s]+$|[\x00-\x08\x0b-\x1f` + commonRange + `]`)
	// starts with a keyword and optionally is followed by a comment
	startsWithKeyword = regexp.MustCompile(`^(true|false|null)\s*((,|\]|\}|#|//|/\*).*)?$`)
	needsEscapeName = regexp.MustCompile(`[,\{\[\}\]\s:#"']|//|/\*`)
}

var meta = map[byte][]byte{
	// table of character substitutions
	'\b': []byte("\\b"),
	'\t': []byte("\\t"),
	'\n': []byte("\\n"),
	'\f': []byte("\\f"),
	'\r': []byte("\\r"),
	'"':  []byte("\\\""),
	'\\': []byte("\\\\"),
}

func (e *hjsonEncoder) quoteReplace(text string) string {
	return string(needsEscape.ReplaceAllFunc([]byte(text), func(a []byte) []byte {
		c := meta[a[0]]
		if c != nil {
			return c
		}
		r, _ := utf8.DecodeRune(a)
		return []byte(fmt.Sprintf("\\u%04x", r))
	}))
}

func (e *hjsonEncoder) quote(value string, separator string, isRootObject bool) {

	// Check if we can insert this string without quotes
	// see hjson syntax (must not parse as true, false, null or number)

	if len(value) == 0 {
		e.WriteString(separator + `""`)
	} else if e.QuoteAlways ||
		needsQuotes.MatchString(value) || (e.QuoteAmbiguousStrings && (startsWithNumber([]byte(value)) ||
		startsWithKeyword.MatchString(value))) {

		// If the string contains no control characters, no quote characters, and no
		// backslash characters, then we can safely slap some quotes around it.
		// Otherwise we first check if the string can be expressed in multiline
		// format or we must replace the offending characters with safe escape
		// sequences.

		if !needsEscape.MatchString(value) {
			e.WriteString(separator + `"` + value + `"`)
		} else if !needsEscapeML.MatchString(value) && !isRootObject {
			e.mlString(value, separator)
		} else {
			e.WriteString(separator + `"` + e.quoteReplace(value) + `"`)
		}
	} else {
		// return without quotes
		e.WriteString(separator + value)
	}
}

func (e *hjsonEncoder) mlString(value string, separator string) {
	a := strings.Split(value, "\n")

	if len(a) == 1 {
		// The string contains only a single line. We still use the multiline
		// format as it avoids escaping the \ character (e.g. when used in a
		// regex).
		e.WriteString(separator + "'''")
		e.WriteString(a[0])
	} else {
		e.writeIndent(e.indent + 1)
		e.WriteString("'''")
		for _, v := range a {
			indent := e.indent + 1
			if len(v) == 0 {
				indent = 0
			}
			e.writeIndent(indent)
			e.WriteString(v)
		}
		e.writeIndent(e.indent + 1)
	}
	e.WriteString("'''")
}

func (e *hjsonEncoder) quoteName(name string) string {
	if len(name) == 0 {
		return `""`
	}

	// Check if we can insert this name without quotes

	if needsEscapeName.MatchString(name) {
		if needsEscape.MatchString(name) {
			name = e.quoteReplace(name)
		}
		return `"` + name + `"`
	}
	// without quotes
	return name
}

type sortAlpha []reflect.Value

func (s sortAlpha) Len() int {
	return len(s)
}
func (s sortAlpha) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortAlpha) Less(i, j int) bool {
	return fmt.Sprintf("%v", s[i]) < fmt.Sprintf("%v", s[j])
}

func (e *hjsonEncoder) writeIndent(indent int) {
	e.WriteString(e.Eol)
	e.WriteString(e.BaseIndentation)
	for i := 0; i < indent; i++ {
		e.WriteString(e.IndentBy)
	}
}

func (e *hjsonEncoder) useMarshalerJSON(
	value reflect.Value,
	noIndent bool,
	separator string,
	isRootObject bool,
) error {
	b, err := value.Interface().(json.Marshaler).MarshalJSON()
	if err != nil {
		return err
	}

	var jsonRoot interface{}
	err = Unmarshal(b, &jsonRoot)
	if err != nil {
		return err
	}

	// Output Hjson with our current options, instead of JSON.
	return e.str(reflect.ValueOf(jsonRoot), noIndent, separator, isRootObject)
}

var marshalerJSON = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
var marshalerText = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()

func (e *hjsonEncoder) str(value reflect.Value, noIndent bool, separator string, isRootObject bool) error {

	// Produce a string from value.

	kind := value.Kind()

	switch kind {
	case reflect.Ptr, reflect.Slice, reflect.Map:
		if e.pDepth++; e.pDepth > depthLimit {
			if e.parents == nil {
				e.parents = map[uintptr]struct{}{}
			}
			p := value.Pointer()
			if _, ok := e.parents[p]; ok {
				return errors.New("Circular reference found, pointer of type " + value.Type().String())
			}
			e.parents[p] = struct{}{}
			defer delete(e.parents, p)
		}
		defer func() { e.pDepth-- }()
	}

	if kind == reflect.Interface || kind == reflect.Ptr {
		if value.IsNil() {
			e.WriteString(separator)
			e.WriteString("null")
			return nil
		}
		return e.str(value.Elem(), noIndent, separator, isRootObject)
	}

	if value.Type().Implements(marshalerJSON) {
		return e.useMarshalerJSON(value, noIndent, separator, isRootObject)
	}

	if value.Type().Implements(marshalerText) {
		b, err := value.Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return err
		}

		return e.str(reflect.ValueOf(string(b)), noIndent, separator, isRootObject)
	}

	switch kind {
	case reflect.String:
		if value.Type() == JSONNumberType {
			n := value.String()
			if n == "" {
				n = "0"
			}
			// without quotes
			e.WriteString(separator + n)
		} else {
			e.quote(value.String(), separator, isRootObject)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.WriteString(separator)
		e.WriteString(strconv.FormatInt(value.Int(), 10))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr:
		e.WriteString(separator)
		e.WriteString(strconv.FormatUint(value.Uint(), 10))

	case reflect.Float32, reflect.Float64:
		// JSON numbers must be finite. Encode non-finite numbers as null.
		e.WriteString(separator)
		number := value.Float()
		if math.IsInf(number, 0) || math.IsNaN(number) {
			e.WriteString("null")
		} else if number == -0 {
			e.WriteString("0")
		} else {
			// find shortest representation ('G' does not work)
			val := strconv.FormatFloat(number, 'f', -1, 64)
			exp := strconv.FormatFloat(number, 'E', -1, 64)
			if len(exp) < len(val) {
				val = strings.ToLower(exp)
			}
			e.WriteString(val)
		}

	case reflect.Bool:
		e.WriteString(separator)
		if value.Bool() {
			e.WriteString("true")
		} else {
			e.WriteString("false")
		}

	case reflect.Slice, reflect.Array:

		len := value.Len()
		if len == 0 {
			e.WriteString(separator)
			e.WriteString("[]")
			break
		}

		indent1 := e.indent
		e.indent++

		if !noIndent && !e.BracesSameLine {
			e.writeIndent(indent1)
		} else {
			e.WriteString(separator)
		}
		e.WriteString("[")

		// Join all of the element texts together, separated with newlines
		for i := 0; i < len; i++ {
			e.writeIndent(e.indent)
			if err := e.str(value.Index(i), true, "", false); err != nil {
				return err
			}
		}

		e.writeIndent(indent1)
		e.WriteString("]")

		e.indent = indent1

	case reflect.Map:
		var fis []fieldInfo
		keys := value.MapKeys()
		sort.Sort(sortAlpha(keys))
		for _, key := range keys {
			fis = append(fis, fieldInfo{
				field: value.MapIndex(key),
				name:  fmt.Sprintf("%v", key),
			})
		}
		return e.writeFields(fis, noIndent, separator, isRootObject)

	case reflect.Struct:
		// Struct field info is identical for all instances of the same type.
		// Only the values on the fields can be different.
		t := value.Type()
		sfis, ok := e.structTypeCache[t]
		if !ok {
			sfis = getStructFieldInfo(t)
			e.structTypeCache[t] = sfis
		}

		// Collect fields first, too see if any should be shown (considering
		// "omitEmpty").
		var fis []fieldInfo
	FieldLoop:
		for _, sfi := range sfis {
			// The field might be found on the root struct or in embedded structs.
			fv := value
			for _, i := range sfi.indexPath {
				if fv.Kind() == reflect.Ptr {
					if fv.IsNil() {
						continue FieldLoop
					}
					fv = fv.Elem()
				}
				fv = fv.Field(i)
			}

			if sfi.omitEmpty && isEmptyValue(fv) {
				continue
			}

			fis = append(fis, fieldInfo{
				field:   fv,
				name:    sfi.name,
				comment: sfi.comment,
			})
		}
		return e.writeFields(fis, noIndent, separator, isRootObject)

	default:
		return errors.New("Unsupported type " + value.Type().String())
	}

	return nil
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	default:
		return false
	}
}

// Marshal returns the Hjson encoding of v using
// default options.
//
// See MarshalWithOptions.
//
func Marshal(v interface{}) ([]byte, error) {
	return MarshalWithOptions(v, DefaultOptions())
}

// MarshalWithOptions returns the Hjson encoding of v.
//
// Marshal traverses the value v recursively.
//
// Boolean values are written as true or false.
//
// Floating point, integer, and json.Number values are written as numbers (with
// decimals only if needed, using . as decimals separator).
//
// String values encode as Hjson strings (quoteless, multiline or
// JSON).
//
// Array and slice values encode as arrays, surrounded by []. Unlike
// json.Marshal, hjson.Marshal will encode a nil-slice as [] instead of null.
//
// Map values encode as objects, surrounded by {}. The map's key type must be
// possible to print to a string. The map keys are sorted alphanumerically and
// used as object keys. Unlike json.Marshal, hjson.Marshal will encode a
// nil-map as {} instead of null.
//
// Struct values also encode as objects, surrounded by {}. Only the exported
// fields are encoded to Hjson. The fields will appear in the same order as in
// the struct.
//
// The encoding of each struct field can be customized by the format string
// stored under the "json" key in the struct field's tag.
// The format string gives the name of the field, possibly followed by a comma
// and "omitempty". The name may be empty in order to specify "omitempty"
// without overriding the default field name.
//
// The "omitempty" option specifies that the field should be omitted
// from the encoding if the field has an empty value, defined as
// false, 0, a nil pointer, a nil interface value, and any empty array,
// slice, map, or string.
//
// As a special case, if the field tag is "-", the field is always omitted.
// Note that a field with name "-" can still be generated using the tag "-,".
//
// Comments can be set on struct fields using the "comment" key in the struct
// field's tag. The comment will be written on the line before the field key,
// prefixed with #. Or possible several lines prefixed by #, if there are line
// breaks (\n) in the comment text.
//
// If both the "json" and the "comment" tag keys are used on a struct field
// they should be separated by whitespace.
//
// Examples of struct field tags and their meanings:
//
//   // Field appears in Hjson as key "myName".
//   Field int `json:"myName"`
//
//   // Field appears in Hjson as key "myName" and the field is omitted from
//   // the object if its value is empty, as defined above.
//   Field int `json:"myName,omitempty"`
//
//   // Field appears in Hjson as key "Field" (the default), but the field is
//   // skipped if empty. Note the leading comma.
//   Field int `json:",omitempty"`
//
//   // Field is ignored by this package.
//   Field int `json:"-"`
//
//   // Field appears in Hjson as key "-".
//   Field int `json:"-,"`
//
//   // Field appears in Hjson preceded by a line just containing `# A comment.`
//   Field int `comment:"A comment."`
//
//   // Field appears in Hjson as key "myName" preceded by a line just
//   // containing `# A comment.`
//   Field int `json:"myName" comment:"A comment."`
//
// Anonymous struct fields are usually marshaled as if their inner exported fields
// were fields in the outer struct, subject to the usual Go visibility rules amended
// as described in the next paragraph.
// An anonymous struct field with a name given in its JSON tag is treated as
// having that name, rather than being anonymous.
// An anonymous struct field of interface type is treated the same as having
// that type as its name, rather than being anonymous.
//
// The Go visibility rules for struct fields are amended for JSON when
// deciding which field to marshal or unmarshal. If there are
// multiple fields at the same level, and that level is the least
// nested (and would therefore be the nesting level selected by the
// usual Go rules), the following extra rules apply:
//
// 1) Of those fields, if any are JSON-tagged, only tagged fields are considered,
// even if there are multiple untagged fields that would otherwise conflict.
//
// 2) If there is exactly one field (tagged or not according to the first rule), that is selected.
//
// 3) Otherwise there are multiple fields, and all are ignored; no error occurs.
//
// Pointer values encode as the value pointed to.
// A nil pointer encodes as the null JSON value.
//
// Interface values encode as the value contained in the interface.
// A nil interface value encodes as the null JSON value.
//
// If an encountered value implements the json.Marshaler interface then the
// function MarshalJSON() is called on it. The JSON is then converted to Hjson
// using the current indentation and options given in the call to json.Marshal().
//
// If an encountered value implements the encoding.TextMarshaler interface
// but not the json.Marshaler interface, then the function MarshalText() is
// called on it to get a text.
//
// Channel, complex, and function values cannot be encoded in Hjson, will
// result in an error.
//
// Hjson cannot represent cyclic data structures and Marshal does not handle
// them. Passing cyclic structures to Marshal will result in an error.
//
func MarshalWithOptions(v interface{}, options EncoderOptions) ([]byte, error) {
	e := &hjsonEncoder{
		indent:          0,
		EncoderOptions:  options,
		structTypeCache: map[reflect.Type][]structFieldInfo{},
	}

	err := e.str(reflect.ValueOf(v), true, e.BaseIndentation, true)
	if err != nil {
		return nil, err
	}
	return e.Bytes(), nil
}
