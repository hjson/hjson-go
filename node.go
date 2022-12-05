package hjson

import (
	"encoding/json"
)

type Comments struct {
	Before string
	Key    string
	Inside string
	After  string
}

type Node struct {
	Value interface{}
	Cm    Comments
}

func (c Node) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Value)
}

func (c *Node) UnmarshalJSON(b []byte) error {
	return Unmarshal(b, c)
}
