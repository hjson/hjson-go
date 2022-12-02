package hjson

import (
	"bytes"
	"encoding/json"
)

type OrderedMap struct {
	k []string
	m map[string]interface{}
}

type KeyValue struct {
	Key   string
	Value interface{}
}

// NewOrderedMap returns a pointer to a new OrderedMap. An OrderedMap should
// always be passed by reference, never by value. If an OrderedMap is passed
// by value then appending new keys won't affect all of the copies of the
// OrderedMap.
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		k: nil,
		m: map[string]interface{}{},
	}
}

// NewOrderedMapFromSlice is like NewOrderedMap but with initial values.
// Example:
//
//	om := NewOrderedMapFromSlice([]KeyValue{
//	 {"B", "first"},
//	 {"A", "second"},
//	})
func NewOrderedMapFromSlice(args []KeyValue) *OrderedMap {
	c := NewOrderedMap()
	for _, elem := range args {
		c.Append(elem.Key, elem.Value)
	}
	return c
}

// Append adds a new key/value pair at the end of the OrderedMap.
func (c *OrderedMap) Append(key string, value interface{}) {
	c.k = append(c.k, key)
	c.m[key] = value
}

func (c *OrderedMap) MarshalJSON() ([]byte, error) {
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
	c.k = nil
	c.m = map[string]interface{}{}
	return Unmarshal(b, c)
}
