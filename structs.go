package hjson

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type fieldInfo struct {
	field   reflect.Value
	name    string
	comment string
}

type structFieldInfo struct {
	name      string
	tagged    bool
	comment   string
	omitEmpty bool
	indexPath []int
}

// dominantField looks through the fields, all of which are known to
// have the same name, to find the single field that dominates the
// others using Go's embedding rules, modified by the presence of
// JSON tags. If there are multiple top-level fields, the boolean
// will be false: This condition is an error in Go and we skip all
// the fields.
func dominantField(fields []structFieldInfo) (structFieldInfo, bool) {
	// The fields are sorted in increasing index-length order, then by presence of tag.
	// That means that the first field is the dominant one. We need only check
	// for error cases: two fields at top level, either both tagged or neither tagged.
	if len(fields) > 1 && len(fields[0].indexPath) == len(fields[1].indexPath) && fields[0].tagged == fields[1].tagged {
		return structFieldInfo{}, false
	}
	return fields[0], true
}

// byIndex sorts by index sequence.
type byIndex []structFieldInfo

func (x byIndex) Len() int { return len(x) }

func (x byIndex) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func (x byIndex) Less(i, j int) bool {
	for k, xik := range x[i].indexPath {
		if k >= len(x[j].indexPath) {
			return false
		}
		if xik != x[j].indexPath[k] {
			return xik < x[j].indexPath[k]
		}
	}
	return len(x[i].indexPath) < len(x[j].indexPath)
}

func getStructFieldInfo(rootType reflect.Type) []structFieldInfo {
	type structInfo struct {
		typ       reflect.Type
		indexPath []int
	}
	var sfis []structFieldInfo
	structsToInvestigate := []structInfo{structInfo{typ: rootType}}
	// Struct types already visited at an earlier depth.
	visited := map[reflect.Type]bool{}
	// Count the number of specific struct types on a specific depth.
	typeDepthCount := map[reflect.Type]int{}

	for len(structsToInvestigate) > 0 {
		curStructs := structsToInvestigate
		structsToInvestigate = []structInfo{}
		curTDC := typeDepthCount
		typeDepthCount = map[reflect.Type]int{}

		for _, curStruct := range curStructs {
			if visited[curStruct.typ] {
				// The struct type has already appeared on an earlier depth. Fields on
				// an earlier depth always have precedence over fields with identical
				// name on a later depth, so no point in investigating this type again.
				continue
			}
			visited[curStruct.typ] = true

			for i := 0; i < curStruct.typ.NumField(); i++ {
				sf := curStruct.typ.Field(i)

				if sf.Anonymous {
					t := sf.Type
					if t.Kind() == reflect.Pointer {
						t = t.Elem()
					}
					// If the field is not exported and not a struct.
					if sf.PkgPath != "" && t.Kind() != reflect.Struct {
						// Ignore embedded fields of unexported non-struct types.
						continue
					}
					// Do not ignore embedded fields of unexported struct types
					// since they may have exported fields.
				} else if sf.PkgPath != "" {
					// Ignore unexported non-embedded fields.
					continue
				}

				jsonTag := sf.Tag.Get("json")
				if jsonTag == "-" {
					continue
				}

				sfi := structFieldInfo{
					name:    sf.Name,
					comment: sf.Tag.Get("comment"),
				}

				splits := strings.Split(jsonTag, ",")
				if splits[0] != "" {
					sfi.name = splits[0]
					sfi.tagged = true
				}
				if len(splits) > 1 {
					for _, opt := range splits[1:] {
						if opt == "omitempty" {
							sfi.omitEmpty = true
						}
					}
				}

				sfi.indexPath = make([]int, len(curStruct.indexPath)+1)
				copy(sfi.indexPath, curStruct.indexPath)
				sfi.indexPath[len(curStruct.indexPath)] = i

				ft := sf.Type
				if ft.Name() == "" && ft.Kind() == reflect.Pointer {
					// Follow pointer.
					ft = ft.Elem()
				}

				// If the current field should be included.
				if sfi.tagged || !sf.Anonymous || ft.Kind() != reflect.Struct {
					sfis = append(sfis, sfi)
					if curTDC[curStruct.typ] > 1 {
						// If there were multiple instances, add a second,
						// so that the annihilation code will see a duplicate.
						// It only cares about the distinction between 1 or 2,
						// so don't bother generating any more copies.
						sfis = append(sfis, sfi)
					}
					continue
				}

				// Record new anonymous struct to explore in next round.
				typeDepthCount[ft]++
				if typeDepthCount[ft] == 1 {
					structsToInvestigate = append(structsToInvestigate, structInfo{
						typ:       ft,
						indexPath: sfi.indexPath,
					})
				}
			}
		}
	}

	sort.Slice(sfis, func(i, j int) bool {
		// sort field by name, breaking ties with depth, then
		// breaking ties with "name came from json tag", then
		// breaking ties with index sequence.
		if sfis[i].name != sfis[j].name {
			return sfis[i].name < sfis[j].name
		}
		if len(sfis[i].indexPath) != len(sfis[j].indexPath) {
			return len(sfis[i].indexPath) < len(sfis[j].indexPath)
		}
		if sfis[i].tagged != sfis[j].tagged {
			return sfis[i].tagged
		}
		return byIndex(sfis).Less(i, j)
	})

	// Delete all fields that are hidden by the Go rules for embedded fields,
	// except that fields with JSON tags are promoted.

	// The fields are sorted in primary order of name, secondary order
	// of field index length. Loop over names; for each name, delete
	// hidden fields by choosing the one dominant field that survives.
	out := sfis[:0]
	for advance, i := 0, 0; i < len(sfis); i += advance {
		// One iteration per name.
		// Find the sequence of sfis with the name of this first field.
		sfi := sfis[i]
		name := sfi.name
		for advance = 1; i+advance < len(sfis); advance++ {
			fj := sfis[i+advance]
			if fj.name != name {
				break
			}
		}
		if advance == 1 { // Only one field with this name
			out = append(out, sfi)
			continue
		}
		dominant, ok := dominantField(sfis[i : i+advance])
		if ok {
			out = append(out, dominant)
		}
	}

	sfis = out
	sort.Sort(byIndex(sfis))

	return sfis
}

func (e *hjsonEncoder) writeFields(
	fis []fieldInfo,
	noIndent bool,
	separator string,
	isRootObject bool,
) error {
	if len(fis) == 0 {
		e.WriteString(separator)
		e.WriteString("{}")
		return nil
	}

	indent1 := e.indent
	if !isRootObject || e.EmitRootBraces {
		if !noIndent && !e.BracesSameLine {
			e.writeIndent(e.indent)
		} else {
			e.WriteString(separator)
		}

		e.indent++
		e.WriteString("{")
	}

	// Join all of the member texts together, separated with newlines
	for i, fi := range fis {
		if len(fi.comment) > 0 {
			for _, line := range strings.Split(fi.comment, e.Eol) {
				if i > 0 || !isRootObject || e.EmitRootBraces {
					e.writeIndent(e.indent)
				}
				e.WriteString(fmt.Sprintf("# %s", line))
			}
		}
		if i > 0 || !isRootObject || e.EmitRootBraces {
			e.writeIndent(e.indent)
		}
		e.WriteString(e.quoteName(fi.name))
		e.WriteString(":")
		if err := e.str(fi.field, false, " ", false); err != nil {
			return err
		}
		if len(fi.comment) > 0 && i < len(fis)-1 {
			e.WriteString(e.Eol)
		}
	}

	if !isRootObject || e.EmitRootBraces {
		e.writeIndent(indent1)
		e.WriteString("}")
	}

	e.indent = indent1

	return nil
}
