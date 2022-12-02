package hjson

import (
	"bytes"
	"encoding/json"
)

type privateOrderedMap struct {
	k []string
	m map[string]interface{}
}

type OrderedMap *privateOrderedMap

type KeyValue struct {
	Key   string
	Value interface{}
}

func NewOrderedMap() OrderedMap {
	return &privateOrderedMap{
		k: nil,
		m: map[string]interface{}{},
	}
}

func NewOrderedMapFromSlice(args []KeyValue) OrderedMap {
	c := NewOrderedMap()
	for _, elem := range args {
		c.Append(elem.Key, elem.Value)
	}
	return c
}

func (c OrderedMap) Append(key string, value interface{}) {
	c.k = append(c.k, key)
	c.m[key] = value
}

func (c OrderedMap) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer

	b.WriteString("{")

	for index, key := range c.k {
		if index > 0 {
			b.WriteString(",")
		}
		jbuf, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		b.Write(jbuf)
		b.WriteString(":")
		jbuf, err = json.Marshal(c.m[key])
		if err != nil {
			return nil, err
		}
		b.Write(jbuf)
	}

	b.WriteString("}")

	return b.Bytes(), nil
}

func (c *OrderedMap) UnmarshalJSON(b []byte) error {
	*c = NewOrderedMap()
	return Unmarshal(b, c)
}
