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

// Node must be used as destination for Unmarshal() or UnmarshalWithOptions()
// whenever comments should be read from the input. The struct is simply a
// wrapper for the actual values and a helper struct containing any comments.
// The Value in the destination Node will be overwritten in the call to
// Unmarshal() or UnmarshalWithOptions(), i.e. node trees are not merged.
// After the unmarshal, Node.Value will contain any of these types:
//
//	nil (no type)
//	float
//	json.Number
//	string
//	bool
//	[]interface{}
//	*hjson.OrderedMap
//
// All elements in an []interface{} or *hjson.OrderedMap will be of the type
// *hjson.Node, so that they can contain comments.
//
// This example shows unmarshalling input with comments, changing the value on
// a single key (the input is assumed to have an object/map as root) and then
// marshalling the node tree again, including comments and with preserved key
// order in the object/map.
//
//	var node hjson.Node
//	err := hjson.Unmarshal(input, &node)
//	if err != nil {
//	  return err
//	}
//	_, err = node.SetKey("setting1", 3)
//	if err != nil {
//	  return err
//	}
//	output, err := hjson.Marshal(node)
//	if err != nil {
//	  return err
//	}
type Node struct {
	Value interface{}
	Cm    Comments
}

// Len returns the length of the value wrapped by this Node, if the value is of
// type *hjson.OrderedMap, []interface{} or string. Otherwise 0 is returned.
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

// AtIndex returns the value (unwrapped from its Node) found at the specified
// index, if this Node contains a value of type *hjson.OrderedMap or
// []interface{}. Returns an error for unexpected types. Panics if index < 0
// or index >= Len().
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
		return nil, fmt.Errorf("Unexpected value type: %v", reflect.TypeOf(c.Value))
	}
	node, ok := elem.(*Node)
	if !ok {
		return nil, fmt.Errorf("Unexpected element type: %v", reflect.TypeOf(elem))
	}
	return node.Value, nil
}

// AtKey returns the value (unwrapped from its Node) found for the specified
// key, if this Node contains a value of type *hjson.OrderedMap. An error is
// returned for unexpected types. The second returned value is true if the key
// was found, false otherwise.
func (c *Node) AtKey(key string) (interface{}, bool, error) {
	if c == nil {
		return nil, false, nil
	}
	om, ok := c.Value.(*OrderedMap)
	if !ok {
		return nil, false, fmt.Errorf("Unexpected value type: %v", reflect.TypeOf(c.Value))
	}
	elem, ok := om.Map[key]
	if !ok {
		return nil, false, nil
	}
	node, ok := elem.(*Node)
	if !ok {
		return nil, false, fmt.Errorf("Unexpected element type: %v", reflect.TypeOf(elem))
	}
	return node.Value, true, nil
}

// Append adds the input value to the end of the []interface{} wrapped by this
// Node. If this Node contains nil without a type, an empty []interface{} is
// first created. If this Node contains a value of any other type, an error is
// returned.
func (c *Node) Append(value interface{}) error {
	if c == nil {
		return fmt.Errorf("Node is nil")
	}
	var arr []interface{}
	if c.Value == nil {
		arr = []interface{}{}
	} else {
		var ok bool
		arr, ok = c.Value.([]interface{})
		if !ok {
			return fmt.Errorf("Unexpected value type: %v", reflect.TypeOf(c.Value))
		}
	}
	c.Value = append(arr, &Node{Value: value})
	return nil
}

// SetIndex assigns the specified value to the child Node found at the specified
// index, if this Node contains a value of type *hjson.OrderedMap or
// []interface{}. Returns an error for unexpected types. Panics if index < 0
// or index >= Len().
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
		return fmt.Errorf("Unexpected value type: %v", reflect.TypeOf(c.Value))
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

// SetKey assigns the specified value to the child Node identified by the
// specified key, if this Node contains a value of the type *hjson.OrderedMap.
// If this Node contains nil without a type, an empty *hjson.OrderedMap is
// first created. If this Node contains a value of any other type or if the
// element idendified by the specified key is not of type *Node, an error is
// returned. If the key cannot be found in the OrderedMap, a new Node is
// created for the specified key, wrapping the specified value. The first
// return value is true if the key already existed in the OrderedMap, false
// otherwise.
func (c *Node) SetKey(key string, value interface{}) (bool, error) {
	if c == nil {
		return false, fmt.Errorf("Node is nil")
	}
	var om *OrderedMap
	if c.Value == nil {
		om = NewOrderedMap()
		c.Value = om
	} else {
		var ok bool
		om, ok = c.Value.(*OrderedMap)
		if !ok {
			return false, fmt.Errorf("Unexpected value type: %v", reflect.TypeOf(c.Value))
		}
	}
	elem, ok := om.Map[key]
	if ok {
		var node *Node
		node, ok = elem.(*Node)
		if ok {
			node.Value = value
		}
	}
	foundKey := true
	if !ok {
		foundKey = om.Set(key, &Node{Value: value})
	}
	return foundKey, nil
}

// NI is an acronym formed from "get Node pointer by Index". Returns the *Node
// element found at the specified index, if this Node contains a value of type
// *hjson.OrderedMap or []interface{}. Returns nil otherwise. Panics if
// index < 0 or index >= Len(). Does not create or alter any value.
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

// NK is an acronym formed from "get Node pointer by Key". Returns the *Node
// element found for the specified key, if this Node contains a value of type
// *hjson.OrderedMap. Returns nil otherwise. Does not create or alter anything.
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

// NKC is an acronym formed from "get Node pointer by Key, Create if not found".
// Returns the *Node element found for the specified key, if this Node contains
// a value of type *hjson.OrderedMap. If this Node contains nil without a type,
// an empty *hjson.OrderedMap is first created. If this Node contains a value of
// any other type or if the element idendified by the specified key is not of
// type *Node, an error is returned. If the key cannot be found in the
// OrderedMap, a new Node is created for the specified key. Example usage:
//
//	var node hjson.Node
//	node.NKC("rootKey1").NKC("subKey1").SetKey("valKey1", "my value")
func (c *Node) NKC(key string) *Node {
	if c == nil {
		return nil
	}
	var om *OrderedMap
	if c.Value == nil {
		om = NewOrderedMap()
		c.Value = om
	} else {
		var ok bool
		om, ok = c.Value.(*OrderedMap)
		if !ok {
			return nil
		}
	}
	if elem, ok := om.Map[key]; ok {
		if node, ok := elem.(*Node); ok {
			return node
		}
	} else {
		node := &Node{}
		om.Set(key, node)
		return node
	}
	return nil
}

// MarshalJSON is an implementation of the json.Marshaler interface, enabling
// hjson.Node trees to be used as input for json.Marshal().
func (c Node) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Value)
}

// UnmarshalJSON is an implementation of the json.Unmarshaler interface,
// enabling hjson.Node to be used as destination for json.Unmarshal().
func (c *Node) UnmarshalJSON(b []byte) error {
	return Unmarshal(b, c)
}
