package hjson

import (
	"bytes"
	"encoding/json"
	"net"
	"reflect"
	"testing"
	"time"
)

func marshalUnmarshalExpected(
	t *testing.T,
	expectedHjson string,
	expectedDst,
	src,
	dst interface{},
) {
	buf, err := Marshal(src)
	if err != nil {
		t.Error(err)
	}
	if string(buf) != expectedHjson {
		t.Errorf("Expected:\n%s\nGot:\n%s\n\n", expectedHjson, string(buf))
	}

	err = Unmarshal(buf, dst)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(expectedDst, dst) {
		t.Errorf("Expected:\n%#v\nGot:\n%#v\n\n", expectedDst, dst)
	}
}

type TestStruct struct {
	A int
	B uint
	C string      `json:"S"`
	D string      `json:",omitempty"`
	E string      `json:"-"`
	F string      `json:"-,"`
	G string      `json:"H,omitempty"`
	J string      `json:"J,omitempty"`
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
		J: "<",
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
	if !bytes.Contains(buf, []byte("J: <\n")) {
		t.Errorf("Missing 'J: <' in marshal output:\n%s", string(buf))
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
	checkKeyValue(t, output, "J", "<")
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
	output = map[string]interface{}{}
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

func TestEmptyMapsAndSlices(t *testing.T) {
	type S2 struct {
		S2Field int
	}

	type S1 struct {
		MapNil        map[string]interface{}
		MapEmpty      map[string]interface{}
		IntSliceNil   []int
		IntSliceEmpty []int
		S2Pointer     *S2
	}
	ts := S1{
		MapEmpty:      map[string]interface{}{},
		IntSliceEmpty: []int{},
	}

	ts2 := map[string]interface{}{
		"MapNil":        map[string]interface{}{},
		"MapEmpty":      map[string]interface{}{},
		"IntSliceNil":   []interface{}{},
		"IntSliceEmpty": []interface{}{},
		"S2Pointer":     nil,
	}

	ds2 := map[string]interface{}{}

	marshalUnmarshalExpected(t, `{
  MapNil: {}
  MapEmpty: {}
  IntSliceNil: []
  IntSliceEmpty: []
  S2Pointer: null
}`, &ts2, &ts, &ds2)

	ts3 := map[string]interface{}{
		"MapNil":        ts.MapNil,
		"MapEmpty":      ts.MapEmpty,
		"IntSliceNil":   ts.IntSliceNil,
		"IntSliceEmpty": ts.IntSliceEmpty,
		"S2Pointer":     ts.S2Pointer,
	}
	ds3 := map[string]interface{}{}

	marshalUnmarshalExpected(t, `{
  IntSliceEmpty: []
  IntSliceNil: []
  MapEmpty: {}
  MapNil: {}
  S2Pointer: null
}`, &ts2, &ts3, &ds3)
}

type TestMarshalStruct struct {
	TestStruct
}

func (s TestMarshalStruct) MarshalJSON() ([]byte, error) {
	return []byte(`"foobar"`), nil
}

func TestEncodeMarshalJSON(t *testing.T) {
	input := TestMarshalStruct{}
	buf, err := Marshal(input)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, []byte(`foobar`)) {
		t.Errorf("Expected '\"foobar\"', got '%s'", string(buf))
	}

	buf, err = Marshal(&input)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, []byte(`foobar`)) {
		t.Errorf("Expected '\"foobar\"', got '%s'", string(buf))
	}

	myMap := map[string]interface{}{
		"Zero": -0,
		"A":    "FirstField",
		"B":    TestMarshalStruct{},
		"C": struct {
			D    string
			Zero int
		}{
			D:    "struct field",
			Zero: -0,
		},
	}
	buf, err = Marshal(&myMap)
	if err != nil {
		t.Error(err)
	}
	expected := "{\n  A: FirstField\n  B: foobar\n  C:\n  {\n    D: struct field\n    Zero: 0\n  }\n  Zero: 0\n}"
	if string(buf) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(buf))
	}
}

func TestEncodeMarshalText(t *testing.T) {
	input := net.ParseIP("127.0.0.1")
	buf, err := Marshal(input)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, []byte(`127.0.0.1`)) {
		t.Errorf("Expected '127.0.0.1', got '%s'", string(buf))
	}
	buf, err = Marshal(&input)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, []byte(`127.0.0.1`)) {
		t.Errorf("Expected '127.0.0.1', got '%s'", string(buf))
	}
}

type TestMarshalInt int

func (s TestMarshalInt) MarshalJSON() ([]byte, error) {
	return []byte(`"foobar"`), nil
}

func TestEncodeMarshalInt(t *testing.T) {
	var input TestMarshalInt
	buf, err := Marshal(input)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, []byte(`foobar`)) {
		t.Errorf("Expected '\"foobar\"', got '%s'", string(buf))
	}
	buf, err = Marshal(&input)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(buf, []byte(`foobar`)) {
		t.Errorf("Expected '\"foobar\"', got '%s'", string(buf))
	}
}

func TestEncodeSliceOfPtrOfPtrOfString(t *testing.T) {
	s := "1"
	s1 := &s
	input := []**string{&s1}
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

func marshalUnmarshal(t *testing.T, input string) {
	buf, err := Marshal(input)
	if err != nil {
		t.Error(err)
		return
	}
	var resultPlain string
	err = Unmarshal(buf, &resultPlain)
	if err != nil {
		t.Error(err)
		return
	}
	if resultPlain != input {
		t.Errorf("Expected: '%v'  Got: '%v'\n", []byte(input), []byte(resultPlain))
	}

	type t_obj struct {
		F string
	}
	obj := t_obj{
		F: input,
	}
	buf, err = Marshal(obj)
	if err != nil {
		t.Error(err)
		return
	}
	var out map[string]interface{}
	err = Unmarshal(buf, &out)
	if err != nil {
		t.Error(err)
		return
	}
	if out["F"] != input {
		t.Errorf("Expected: '%v'  Got: '%v'\n", []byte(input), []byte(out["F"].(string)))
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	marshalUnmarshal(t, "0\r'")
	marshalUnmarshal(t, "0\r")
	marshalUnmarshal(t, "0\n'")
	marshalUnmarshal(t, "0\n")
	marshalUnmarshal(t, "\t0\na\tb\t")
	marshalUnmarshal(t, "\t0\n\tab")
	marshalUnmarshal(t, "0\r\n'")
	marshalUnmarshal(t, "0\r\n")
}

func TestCircularReference(t *testing.T) {
	timeout := time.After(3 * time.Second)
	done := make(chan bool)
	go func() {
		type Node struct {
			Self *Node
		}
		var obj Node
		obj.Self = &obj
		_, err := Marshal(obj)
		if err == nil {
			t.Error("No error returned for circular reference")
		}
		done <- true
	}()

	select {
	case <-timeout:
		t.Error("The circular reference test is taking too long, is probably stuck in an infinite loop.")
	case <-done:
	}
}

func TestPrivateStructFields(t *testing.T) {
	obj := struct {
		somePrivateField string
	}{
		"TEST",
	}
	b, err := Marshal(obj)
	if err != nil {
		t.Error(err)
		return
	}
	if string(b) != "{}" {
		t.Errorf("Expected '{}', got '%s'", string(b))
	}
}

func TestMarshalDuplicateFields(t *testing.T) {
	type A struct {
		B int      `json:"rate"`
		C []string `json:"rate"`
	}

	a := A{
		B: 3,
		C: []string{"D", "E"},
	}

	buf, err := Marshal(&a)
	if err != nil {
		t.Error(err)
	}
	expected := `{
  rate: 3
  rate:
  [
    D
    E
  ]
}`
	if string(buf) != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s\n", expected, string(buf))
	}

	var a2 A
	err = Unmarshal([]byte("rate: 5"), &a2)
	if err != nil {
		t.Error(err)
	}
	// json.Unmarshal will not write at all to fields with duplicate names.
	if !reflect.DeepEqual(a2, A{}) {
		t.Errorf("Expected empty struct")
	}

	type B struct {
		B int `json:"rate"`
	}
	var b B
	err = Unmarshal([]byte("rate: 5"), &b)
	if err != nil {
		t.Error(err)
	}
	if b.B != 5 {
		t.Errorf("Expected 5, got %d\n", b.B)
	}
}

func TestMarshalMapIntKey(t *testing.T) {
	m := map[int]bool{
		3: true,
	}
	buf, err := Marshal(m)
	if err != nil {
		t.Error(err)
	}
	expected := `{
  3: true
}`
	if string(buf) != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s\n", expected, string(buf))
	}

	m2 := map[int]bool{}
	err = Unmarshal(buf, &m2)
	if err != nil {
		t.Error(err)
	}
	if !m2[3] {
		t.Errorf("Failed to unmarshal into map with int key")
	}
}

func TestMarshalJsonNumber(t *testing.T) {
	var n json.Number
	buf, err := Marshal(n)
	if err != nil {
		t.Error(err)
	}
	expected := `0`
	if string(buf) != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s\n", expected, string(buf))
	}

	n = json.Number("3e5")
	buf, err = Marshal(n)
	if err != nil {
		t.Error(err)
	}
	expected = `3e5`
	if string(buf) != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s\n", expected, string(buf))
	}
}
