package hjson

import (
	"bytes"
	"encoding/json"
)

type keyVal struct {
	key   string
	value interface{}
}

type orderedMap []keyVal

func (c orderedMap) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer

	b.WriteString("{")

	for index, elem := range c {
		if index > 0 {
			b.WriteString(",")
		}
		jbuf, err := json.Marshal(elem.key)
		if err != nil {
			return nil, err
		}
		b.Write(jbuf)
		b.WriteString(":")
		jbuf, err = json.Marshal(elem.value)
		if err != nil {
			return nil, err
		}
		b.Write(jbuf)
	}

	b.WriteString("}")

	return b.Bytes(), nil
}
