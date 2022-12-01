package hjson

import (
	"bytes"
	"encoding/json"
)

type OrderedMap struct {
	Keys *[]string
	Map  map[string]interface{}
}

type KeyValue struct {
	Key   string
	Value interface{}
}

func (c *OrderedMap) initIfNeeded() {
	if c.Keys == nil {
		c.Keys = &[]string{}
		c.Map = map[string]interface{}{}
	}
}

func CreateOrderedMap(args []KeyValue) OrderedMap {
	var c OrderedMap
	for _, elem := range args {
		c.Append(elem.Key, elem.Value)
	}
	return c
}

func (c *OrderedMap) Append(key string, value interface{}) {
	c.initIfNeeded()
	*c.Keys = append(*c.Keys, key)
	c.Map[key] = value
}

func (c OrderedMap) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer

	b.WriteString("{")

	if c.Keys != nil {
		for index, key := range *c.Keys {
			if index > 0 {
				b.WriteString(",")
			}
			jbuf, err := json.Marshal(key)
			if err != nil {
				return nil, err
			}
			b.Write(jbuf)
			b.WriteString(":")
			jbuf, err = json.Marshal(c.Map[key])
			if err != nil {
				return nil, err
			}
			b.Write(jbuf)
		}
	}

	b.WriteString("}")

	return b.Bytes(), nil
}
