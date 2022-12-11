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
