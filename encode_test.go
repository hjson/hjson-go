package hjson

import (
	"reflect"
	"testing"
)

type TestStruct struct {
	A int
	B uint
	C string      `json:"S"`
	D string      `json:",omitempty"`
	E string      `json:"-"`
	F string      `json:"-,"`
	G string      `json:"H,omitempty"`
	U int         `json:",omitempty"`
	V uint        `json:",omitempty"`
	W float32     `json:",omitempty"`
	X bool        `json:",omitempty"`
	Y []int       `json:",omitempty"`
	Z *TestStruct `json:",omitempty"`
}

func TestEncodeEmptyStruct(t *testing.T) {
	buf, err := Marshal(struct{}{})
	if err != nil {
		t.Error(err)
	}
	if string(buf) != "{}" {
		t.Error("Empty struct encoding error")
	}
}

func TestEncodeStruct(t *testing.T) {
	var output map[string]interface{}
	input := TestStruct{
		A: 1,
		B: 2,
		C: "foo",
		D: "bar",
		E: "baz",
		F: "qux",
		G: "thud",
		U: 3,
		V: 4,
		W: 5.0,
		X: true,
		Y: []int{1, 2, 3},
		Z: &TestStruct{},
	}
	buf, err := Marshal(input)
	if err != nil {
		t.Error(err)
	}
	err = Unmarshal(buf, &output)
	if err != nil {
		t.Error(err)
	}
	checkKeyValue(t, output, "A", 1.0)
	checkKeyValue(t, output, "B", 2.0)
	checkKeyValue(t, output, "S", "foo")
	checkKeyValue(t, output, "D", "bar")
	checkKeyValue(t, output, "-", "qux")
	checkKeyValue(t, output, "H", "thud")
	checkMissing(t, output, "C")
	checkMissing(t, output, "E")
	checkMissing(t, output, "F")
	checkKeyValue(t, output, "U", 3.0)
	checkKeyValue(t, output, "V", 4.0)
	checkKeyValue(t, output, "W", 5.0)
	checkKeyValue(t, output, "X", true)
	checkKeyValue(t, output, "Y", []interface{}{1.0, 2.0, 3.0})
	checkKeyValue(t, output, "Z", map[string]interface{}{
		"A": 0.0,
		"B": 0.0,
		"S": "",
		"-": "",
	})
	input.D = ""
	input.G = ""
	input.U = 0
	input.V = 0
	input.W = 0.0
	input.X = false
	input.Y = nil
	input.Z = nil
	buf, err = Marshal(input)
	if err != nil {
		t.Error(err)
	}
	err = Unmarshal(buf, &output)
	if err != nil {
		t.Error(err)
	}
	checkKeyValue(t, output, "A", 1.0)
	checkKeyValue(t, output, "B", 2.0)
	checkMissing(t, output, "C")
	checkMissing(t, output, "D")
	checkMissing(t, output, "E")
	checkMissing(t, output, "G")
	checkMissing(t, output, "H")
	checkMissing(t, output, "U")
	checkMissing(t, output, "V")
	checkMissing(t, output, "W")
	checkMissing(t, output, "X")
	checkMissing(t, output, "Y")
	checkMissing(t, output, "Z")
}

func checkKeyValue(t *testing.T, m map[string]interface{}, key string, exp interface{}) {
	obs, ok := m[key]
	if !ok {
		t.Error("Missing key", key)
	}
	if !reflect.DeepEqual(exp, obs) {
		t.Errorf("%v != %v", exp, obs)
	}
}

func checkMissing(t *testing.T, m map[string]interface{}, key string) {
	_, ok := m[key]
	if ok {
		t.Error("Unexpected key", key)
	}
}

type TestMarshalStruct struct {
	TestStruct
}

func (s TestMarshalStruct) MarshalJSON() ([]byte, error) {
	return []byte(`"foobar"`), nil
}

func TestEncodeMarshal(t *testing.T) {
	input := TestMarshalStruct{}
	buf, err := Marshal(input)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, []byte(`"foobar"`)) {
		t.Error("Marshaler interface error")
	}
	buf, err = Marshal(&input)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, []byte(`"foobar"`)) {
		t.Error("Marshaler interface error")
	}
}

func TestEncodeSliceOfPtrOfPtrOfString(t *testing.T) {
	s:="1"
	s1:=&s
	input:=[]**string{&s1}
	buf, err := Marshal(input)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(buf, []byte(`[
  "1"
]`)) {
		t.Error("Marshaler interface error")
	}
}

func TestNoRootBraces(t *testing.T) {
	input := struct {
		Foo string
	}{
		Foo: "Bar",
	}
	opt := DefaultOptions()
	opt.EmitRootBraces = false
	buf, err := MarshalWithOptions(input, opt)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, []byte(`Foo: Bar`)) {
		t.Error("Encode struct with EmitRootBraces false")
	}

	theMap := map[string]interface{}{
		"Foo": "Bar",
	}
	buf, err = MarshalWithOptions(theMap, opt)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, []byte(`Foo: Bar`)) {
		t.Error("Encode map with EmitRootBraces false")
	}
}

func TestBaseIndentation(t *testing.T) {
	input := struct {
		Foo string
	}{
		Foo: "Bar",
	}
	facit := []byte(`   {
     Foo: Bar
   }`)
	opt := DefaultOptions()
	opt.BaseIndentation = "   "
	buf, err := MarshalWithOptions(input, opt)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, facit) {
		t.Error("Encode with BaseIndentation, comparison:\n", string(buf), "\n", string(facit))
	}
}

func TestQuoteAmbiguousStrings(t *testing.T) {
	theMap := map[string]interface{}{
		"One":   "1",
		"Null":  "null",
		"False": "false",
	}
	facit := []byte(`{
  False: false
  Null: null
  One: 1
}`)
	opt := DefaultOptions()
	opt.QuoteAlways = false
	opt.QuoteAmbiguousStrings = false
	buf, err := MarshalWithOptions(theMap, opt)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, facit) {
		t.Error("Encode with QuoteAmbiguousStrings false, comparison:\n", string(buf), "\n", string(facit))
	}
}
