package hjson

import "bytes"

type keyVal struct {
	key   []byte
	value []byte
}

type orderedMap []keyVal

func (c orderedMap) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer

	b.WriteString("{")

	for index, elem := range c {
		if index > 0 {
			b.WriteString(",")
		}
		b.WriteString(`"` + string(elem.key) + `":"` + string(elem.value) + `"`)
	}

	b.WriteString("}")

	return b.Bytes(), nil
}
