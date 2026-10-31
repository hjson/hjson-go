package hjson

import (
	"encoding/json"
	"reflect"
	"testing"
)

func verifyContent(t *testing.T, om *OrderedMap, txtExpected string) {
	bOut, err := json.Marshal(om)
	if err != nil {
		t.Error(err)
	}

	if string(bOut) != txtExpected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s\n\n", txtExpected, string(bOut))
	}
}

func TestAppend(t *testing.T) {
	om := NewOrderedMap()
	om.Set("B", "first")
	om.Set("A", 2)

	verifyContent(t, om, `{"B":"first","A":2}`)
}

func TestInsert(t *testing.T) {
	om := NewOrderedMapFromSlice([]KeyValue{
		{"B", "first"},
		{"A", 2},
	})

	_, ok := om.Insert(1, "C", 1)
	if ok {
		t.Error("Insert returned true for non-existing key")
	}
	verifyContent(t, om, `{"B":"first","C":1,"A":2}`)

	oldVal, ok := om.Insert(3, "C", 3)
	if !ok {
		t.Error("Insert returned false for existing key")
	}
	if oldVal != 1 {
		t.Errorf("Expected old value 1, got: '%v'", oldVal)
	}
	verifyContent(t, om, `{"B":"first","C":3,"A":2}`)

	if om.Len() != 3 {
		t.Errorf("Expected length: 3  Got length: %d\n", om.Len())
	}

	if om.AtIndex(1) != 3 {
		t.Errorf("Expected value 3 at index 1.  Got value: %d\n", om.AtIndex(3))
	}

	if om.Map["C"] != 3 {
		t.Errorf("Expected value 3 for key C.  Got value: %d\n", om.AtIndex(3))
	}

	oldVal, found := om.DeleteKey("XYZ")
	if found {
		t.Errorf("DeleteKey returned true for non-existing key.")
	}

	oldVal, found = om.DeleteKey("C")
	if !found {
		t.Errorf("DeleteKey returned false for existing key.")
	}
	if oldVal != 3 {
		t.Errorf("Expected old value 3, got: '%v'", oldVal)
	}
	verifyContent(t, om, `{"B":"first","A":2}`)

	key, oldVal := om.DeleteIndex(1)
	if key != "A" {
		t.Errorf("Expected key 'A', got: '%v'", key)
	}
	if oldVal != 2 {
		t.Errorf("Expected old value 2, got: '%v'", oldVal)
	}
	verifyContent(t, om, `{"B":"first"}`)

	key, oldVal = om.DeleteIndex(0)
	if key != "B" {
		t.Errorf("Expected key 'B', got: '%v'", key)
	}
	if oldVal != "first" {
		t.Errorf("Expected old value 'first', got: '%v'", oldVal)
	}
	verifyContent(t, om, `{}`)
}

func TestUnmarshalJSON(t *testing.T) {
	var om *OrderedMap
	err := json.Unmarshal([]byte(`{"B":"first","C":3,"sub":{"z":7,"y":8},"A":2}`), &om)
	if err != nil {
		t.Error(err)
	}

	verifyContent(t, om, `{"B":"first","C":3,"sub":{"z":7,"y":8},"A":2}`)

	if _, ok := om.Map["C"].(float64); !ok {
		t.Errorf("Expected type float64, got type %v", reflect.TypeOf(om.Map["C"]))
	}
}

func TestUnmarshalJSON_2(t *testing.T) {
	var om OrderedMap
	err := json.Unmarshal([]byte(`{"B":"first","C":3,"sub":{"z":7,"y":8},"A":2}`), &om)
	if err != nil {
		t.Error(err)
	}

	verifyContent(t, &om, `{"B":"first","C":3,"sub":{"z":7,"y":8},"A":2}`)
}

func TestUnmarshalHJSON(t *testing.T) {
	var om *OrderedMap
	err := Unmarshal([]byte(`{"B":"first","C":3,"sub":{"z":7,"y":8},"A":2}`), &om)
	if err != nil {
		t.Error(err)
	}

	verifyContent(t, om, `{"B":"first","C":3,"sub":{"z":7,"y":8},"A":2}`)

	if _, ok := om.Map["C"].(float64); !ok {
		t.Errorf("Expected type float64, got type %v", reflect.TypeOf(om.Map["C"]))
	}
}

func TestUnmarshalHJSON_2(t *testing.T) {
	var om OrderedMap
	err := Unmarshal([]byte(`{"B":"first","C":3,"sub":{"z":7,"y":8},"A":2}`), &om)
	if err != nil {
		t.Error(err)
	}

	verifyContent(t, &om, `{"B":"first","C":3,"sub":{"z":7,"y":8},"A":2}`)
}

func TestZeroValueOrderedMap(t *testing.T) {
	// An OrderedMap that was not made by NewOrderedMap() has no map to put
	// values in. Insert() and Set(), which is Insert() at the end, create one.
	// Every other operation on such an OrderedMap worked already.

	t.Run("Set", func(t *testing.T) {
		var om OrderedMap
		if oldVal, found := om.Set("B", "first"); oldVal != nil || found {
			t.Error("Set returned a value for a key that was not there")
		}
		if oldVal, found := om.Set("A", 2); oldVal != nil || found {
			t.Error("Set returned a value for a key that was not there")
		}
		if oldVal, found := om.Set("B", "second"); oldVal != "first" || !found {
			t.Errorf("Set returned %v, %v; want first, true", oldVal, found)
		}
		verifyContent(t, &om, `{"B":"second","A":2}`)
	})

	t.Run("Insert", func(t *testing.T) {
		om := &OrderedMap{}
		om.Insert(0, "B", "first")
		om.Insert(0, "A", 2)
		verifyContent(t, om, `{"A":2,"B":"first"}`)
	})

	t.Run("as a struct field", func(t *testing.T) {
		var holder struct {
			Meta OrderedMap `json:"meta"`
		}
		holder.Meta.Set("a", 1)
		b, err := Marshal(holder)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != "{\n  meta: {\n    a: 1\n  }\n}" {
			t.Errorf("Expected:\n%s\n\nGot:\n%s\n", "{\n  meta: {\n    a: 1\n  }\n}", b)
		}
	})

	t.Run("keys given without a map", func(t *testing.T) {
		om := OrderedMap{Keys: []string{"a"}}
		om.Set("b", 2)
		if !reflect.DeepEqual(om.Keys, []string{"a", "b"}) {
			t.Errorf("Keys is %v; want [a b]", om.Keys)
		}
		if v, found := om.AtKey("b"); v != 2 || !found {
			t.Errorf("AtKey returned %v, %v; want 2, true", v, found)
		}
		// The map says which keys exist, the Keys slice only their order, so a
		// key that is only in Keys still reads as missing.
		if v, found := om.AtKey("a"); v != nil || found {
			t.Errorf("AtKey returned %v, %v; want nil, false", v, found)
		}
	})

	t.Run("delete after insert", func(t *testing.T) {
		var om OrderedMap
		om.Set("a", 1)
		if v, found := om.DeleteKey("a"); v != 1 || !found {
			t.Errorf("DeleteKey returned %v, %v; want 1, true", v, found)
		}
		if om.Len() != 0 {
			t.Errorf("Len is %d; want 0", om.Len())
		}
		verifyContent(t, &om, `{}`)
	})

	t.Run("reading does not create a map", func(t *testing.T) {
		var om OrderedMap
		om.Len()
		om.AtKey("a")
		om.DeleteKey("a")
		if _, err := Marshal(&om); err != nil {
			t.Fatal(err)
		}
		if om.Map != nil {
			t.Errorf("Map was made by a read: %v", om.Map)
		}
	})

	t.Run("through a Node", func(t *testing.T) {
		n := Node{Value: &OrderedMap{}}
		if _, found, err := n.SetKey("a", 1); err != nil || found {
			t.Errorf("SetKey returned %v; %v; want false, nil", found, err)
		}
		b, err := Marshal(n)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != "{\n  a: 1\n}" {
			t.Errorf("Got:\n%s\n", b)
		}
	})

	t.Run("index out of range still panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("Insert did not panic for an index past the end")
			}
		}()
		om := &OrderedMap{}
		om.Insert(1, "a", 1)
	})
}
