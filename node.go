package hjson

import (
	"encoding/json"
	"fmt"
	"reflect"
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

func (c *Node) Len() int {
	if c == nil {
		return 0
	}
	switch cont := c.Value.(type) {
	case *OrderedMap:
		return cont.Len()
	case []interface{}:
		return len(cont)
	case string:
		return len(cont)
	}
	return 0
}

func (c *Node) AtIndex(index int) (interface{}, error) {
	if c == nil {
		return nil, fmt.Errorf("Node is nil")
	}
	var elem interface{}
	switch cont := c.Value.(type) {
	case *OrderedMap:
		elem = cont.AtIndex(index)
	case []interface{}:
		elem = cont[index]
	default:
		return nil, fmt.Errorf("Unexpected value type: %v", reflect.ValueOf(c.Value))
	}
	node, ok := elem.(*Node)
	if !ok {
		return nil, fmt.Errorf("Unexpected element type: %v", reflect.ValueOf(elem))
	}
	return node.Value, nil
}

func (c *Node) AtKey(key string) (interface{}, bool, error) {
	if c == nil {
		return nil, false, nil
	}
	om, ok := c.Value.(*OrderedMap)
	if !ok {
		return nil, false, fmt.Errorf("Unexpected value type: %v", reflect.ValueOf(c.Value))
	}
	elem, ok := om.Map[key]
	if !ok {
		return nil, false, nil
	}
	node, ok := elem.(*Node)
	if !ok {
		return nil, false, fmt.Errorf("Unexpected element type: %v", reflect.ValueOf(elem))
	}
	return node.Value, true, nil
}

func (c *Node) SetIndex(index int, value interface{}) error {
	if c == nil {
		return fmt.Errorf("Node is nil")
	}
	var elem interface{}
	switch cont := c.Value.(type) {
	case *OrderedMap:
		elem = cont.AtIndex(index)
	case []interface{}:
		elem = cont[index]
	default:
		return fmt.Errorf("Unexpected value type: %v", reflect.ValueOf(c.Value))
	}
	node, ok := elem.(*Node)
	if ok {
		node.Value = value
	} else {
		switch cont := c.Value.(type) {
		case *OrderedMap:
			cont.Map[cont.Keys[index]] = &Node{Value: value}
		case []interface{}:
			cont[index] = &Node{Value: value}
		}
	}
	return nil
}

func (c *Node) SetKey(key string, value interface{}) error {
	if c == nil {
		return fmt.Errorf("Node is nil")
	}
	om, ok := c.Value.(*OrderedMap)
	if !ok {
		return fmt.Errorf("Unexpected value type: %v", reflect.ValueOf(c.Value))
	}
	elem, ok := om.Map[key]
	if ok {
		var node *Node
		node, ok = elem.(*Node)
		if ok {
			node.Value = value
		}
	}
	if !ok {
		om.Map[key] = &Node{Value: value}
	}
	return nil
}

func (c *Node) NI(index int) *Node {
	if c == nil {
		return nil
	}
	var elem interface{}
	switch cont := c.Value.(type) {
	case *OrderedMap:
		elem = cont.AtIndex(index)
	case []interface{}:
		elem = cont[index]
	default:
		return nil
	}
	if node, ok := elem.(*Node); ok {
		return node
	}
	return nil
}

func (c *Node) NK(key string) *Node {
	if c == nil {
		return nil
	}
	om, ok := c.Value.(*OrderedMap)
	if !ok {
		return nil
	}
	if elem, ok := om.Map[key]; ok {
		if node, ok := elem.(*Node); ok {
			return node
		}
	}
	return nil
}

func (c Node) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Value)
}

func (c *Node) UnmarshalJSON(b []byte) error {
	return Unmarshal(b, c)
}
