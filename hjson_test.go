package hjson

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fixEOL(data []byte) []byte {
	// The output from Marshal() always uses Unix EOL, but git might have
	// converted files to Windows EOL on Windows, therefore we convert all
	// "\r\n" to "\n".
	return bytes.Replace(data, []byte("\r\n"), []byte("\n"), -1)
}

func getContent(file string) []byte {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	return fixEOL(data)
}

func getTestContent(name string) []byte {
	p := fmt.Sprintf("./assets/%s_test.hjson", name)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		p = fmt.Sprintf("./assets/%s_test.json", name)
	}
	return getContent(p)
}

func getResultContent(name string) ([]byte, []byte, []byte, []byte) {
	p1 := fmt.Sprintf("./assets/sorted/%s_result.json", name)
	p2 := fmt.Sprintf("./assets/sorted/%s_result.hjson", name)
	p3 := fmt.Sprintf("./assets/comments2/%s_result.hjson", name)
	p4 := fmt.Sprintf("./assets/comments3/%s_result.hjson", name)
	return getContent(p1), getContent(p2), getContent(p3), getContent(p4)
}

func fixJSON(data []byte) []byte {
	data = bytes.Replace(data, []byte("\\u003c"), []byte("<"), -1)
	data = bytes.Replace(data, []byte("\\u003e"), []byte(">"), -1)
	data = bytes.Replace(data, []byte("\\u0026"), []byte("&"), -1)
	data = bytes.Replace(data, []byte("\\u0008"), []byte("\\b"), -1)
	data = bytes.Replace(data, []byte("\\u000c"), []byte("\\f"), -1)
	return data
}

func run(t *testing.T, file string) {
	name := strings.TrimSuffix(file, "_test"+filepath.Ext(file))
	t.Logf("running %s", name)
	shouldFail := strings.HasPrefix(file, "fail")

	testContent := getTestContent(name)
	var data interface{}
	if err := Unmarshal(testContent, &data); err != nil {
		if !shouldFail {
			t.Error(err)
		}
		return
	} else if shouldFail {
		t.Error(errors.New(name + " should_fail!"))
		return
	}

	rjson, rhjson, cm2, cm3 := getResultContent(name)

	actualHjson, err := Marshal(data)
	if err != nil {
		t.Error(err)
		return
	}
	actualHjson = append(actualHjson, '\n')
	actualJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Error(err)
		return
	}
	actualJSON = append(actualJSON, '\n')
	actualJSON = fixJSON(actualJSON)
	var actualCm2 []byte
	{
		var node Node
		decOpt := DefaultDecoderOptions()
		decOpt.WhitespaceAsComments = false
		if err := UnmarshalWithOptions(testContent, &node, decOpt); err != nil {
			t.Error(err)
			return
		}
		actualCm2, err = Marshal(node)
		if err != nil {
			t.Error(err)
			return
		}
		if len(actualCm2) > 0 && actualCm2[len(actualCm2)-1] != '\n' {
			actualCm2 = append(actualCm2, '\n')
		}
	}
	var actualCm3 []byte
	{
		var node Node
		if err := Unmarshal(testContent, &node); err != nil {
			t.Error(err)
			return
		}
		actualCm3, err = Marshal(node)
		if err != nil {
			t.Error(err)
			return
		}
		if len(actualCm3) > 0 && actualCm3[len(actualCm3)-1] != '\n' {
			actualCm3 = append(actualCm3, '\n')
		}
	}

	hjsonOK := bytes.Equal(rhjson, actualHjson)
	jsonOK := bytes.Equal(rjson, actualJSON)
	cm2OK := bytes.Equal(cm2, actualCm2)
	cm3OK := bytes.Equal(cm3, actualCm3)
	if !hjsonOK {
		t.Errorf("%s\n---hjson expected\n%s\n---hjson actual\n%s\n---\n", name, rhjson, actualHjson)
		//		err = ioutil.WriteFile(fmt.Sprintf("./assets/sorted/%s_result.hjson", name), actualHjson, 0644)
		//		if err != nil {
		//			t.Error(err)
		//		}
	}
	if !jsonOK {
		t.Errorf("%s\n---json expected\n%s\n---json actual\n%s\n---\n", name, rjson, actualJSON)
		//		err = ioutil.WriteFile(fmt.Sprintf("./assets/sorted/%s_result.json", name), actualJSON, 0644)
		//		if err != nil {
		//			t.Error(err)
		//		}
	}
	if !cm2OK {
		t.Errorf("%s\n---cm2 expected\n%s\n---cm2 actual\n%s\n---\n", name, cm2, actualCm2)
		//		err = ioutil.WriteFile(fmt.Sprintf("./assets/comments2/%s_result.hjson", name), actualCm2, 0644)
		//		if err != nil {
		//			t.Error(err)
		//		}
	}
	{
		var roundTrip interface{}
		err = Unmarshal(actualCm2, &roundTrip)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(data, roundTrip) {
			t.Errorf("cm2 roundtrip failed!")
		}
	}
	if !cm3OK {
		t.Errorf("%s\n---cm3 expected\n%s\n---cm3 actual\n%s\n---\n", name, cm3, actualCm3)
		//		err = ioutil.WriteFile(fmt.Sprintf("./assets/comments3/%s_result.hjson", name), actualCm3, 0644)
		//		if err != nil {
		//			t.Error(err)
		//		}
	}
	{
		var roundTrip interface{}
		err = Unmarshal(actualCm3, &roundTrip)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(data, roundTrip) {
			t.Errorf("cm3 roundtrip failed!")
		}
	}
}

func TestHjson(t *testing.T) {

	files := strings.Split(string(getContent("assets/testlist.txt")), "\n")

	for _, file := range files {
		if file != "" && !strings.HasPrefix(file, "stringify/quotes") && !strings.HasPrefix(file, "extra/") {
			run(t, file)
		}
	}
}

func TestInvalidDestinationType(t *testing.T) {
	input := []byte(`[1,2,3,4]`)
	var dat map[string]interface{}
	err := Unmarshal(input, &dat)
	if err == nil {
		t.Errorf("Should have failed when trying to unmarshal an array to a map.")
	}

	err = Unmarshal(input, 3)
	if err == nil {
		t.Errorf("Should have failed when trying to unmarshal into non-pointer.")
	}
}

func TestStructDestinationType(t *testing.T) {
	var obj struct {
		A int
		B int
		C string
		D string
	}
	err := Unmarshal([]byte("A: 1\nB:2\nC: \u003c\nD: <"), &obj)
	if err != nil {
		t.Error(err)
	}
	if obj.A != 1 || obj.B != 2 || obj.C != "<" || obj.D != "<" {
		t.Errorf("Unexpected obj values: %+v", obj)
	}
}

func TestNilValue(t *testing.T) {
	var dat interface{}
	err := Unmarshal([]byte(`[1,2,3,4]`), dat)
	if err == nil {
		panic("Passing v = <nil> to Unmarshal should return an error")
	}
}

func TestReadmeUnmarshalToStruct(t *testing.T) {
	type Sample struct {
		Rate  int
		Array []string
	}

	type SampleAlias struct {
		Rett    int      `json:"rate"`
		Ashtray []string `json:"array"`
	}

	sampleText := []byte(`
{
	# specify rate in requests/second
	rate: 1000
	array:
	[
		foo
		bar
	]
}`)

	{
		var sample Sample
		Unmarshal(sampleText, &sample)
		if sample.Rate != 1000 || sample.Array[0] != "foo" {
			t.Errorf("Unexpected sample values: %+v", sample)
		}
	}

	{
		var sampleAlias SampleAlias
		Unmarshal(sampleText, &sampleAlias)
		if sampleAlias.Rett != 1000 || sampleAlias.Ashtray[0] != "foo" {
			t.Errorf("Unexpected sampleAlias values: %+v", sampleAlias)
		}
	}
}

func TestUnknownFields(t *testing.T) {
	v := struct {
		B string
		C int
	}{}
	b := []byte("B: b\nC: 3\nD: 4\n")
	err := Unmarshal(b, &v)
	if err != nil {
		t.Error(err)
	}
	err = UnmarshalWithOptions(b, &v, DecoderOptions{DisallowUnknownFields: true})
	if err == nil {
		t.Errorf("Should have returned error for unknown field D")
	}
}

type testOrderedMapA struct {
	*OrderedMap
}

func (c *testOrderedMapA) UnmarshalJSON(in []byte) error {
	c.OrderedMap = NewOrderedMap()
	index := 0
	for true {
		i1 := bytes.IndexByte(in[index:], '"')
		if i1 < 0 {
			break
		}
		i1 += index
		index = i1 + 1

		i2 := bytes.IndexByte(in[index:], '"')
		if i2 < 0 {
			break
		}
		i2 += index
		index = i2 + 1

		i3 := bytes.IndexByte(in[index:], '"')
		if i3 < 0 {
			break
		}
		i3 += index
		index = i3 + 1

		i4 := bytes.IndexByte(in[index:], '"')
		if i4 < 0 {
			break
		}
		i4 += index
		index = i4 + 1

		c.OrderedMap.Set(
			string(in[i1+1:i2]),
			string(in[i3+1:i4]),
		)
	}

	return nil
}

func (c testOrderedMapA) ElemType() reflect.Type {
	return reflect.TypeOf("")
}

type testOrderedMapB struct {
	*OrderedMap
}

func (c *testOrderedMapB) UnmarshalJSON(in []byte) error {
	c.OrderedMap = NewOrderedMap()
	index := 0
	for true {
		i1 := bytes.IndexByte(in[index:], '"')
		if i1 < 0 {
			break
		}
		i1 += index
		index = i1 + 1

		i2 := bytes.IndexByte(in[index:], '"')
		if i2 < 0 {
			break
		}
		i2 += index
		index = i2 + 1

		i3 := bytes.IndexByte(in[index:], '"')
		if i3 < 0 {
			break
		}
		i3 += index
		index = i3 + 1

		i4 := bytes.IndexByte(in[index:], '"')
		if i4 < 0 {
			break
		}
		i4 += index
		index = i4 + 1

		c.OrderedMap.Set(
			string(in[i1+1:i2]),
			string(in[i3+1:i4]),
		)
	}

	return nil
}

func (c *testOrderedMapB) ElemType() reflect.Type {
	return reflect.TypeOf("")
}

type testSliceElemTyperA []interface{}

func (c testSliceElemTyperA) ElemType() reflect.Type {
	return reflect.TypeOf("")
}

type testSliceElemTyperB []interface{}

func (c *testSliceElemTyperB) ElemType() reflect.Type {
	return reflect.TypeOf("")
}

func TestUnmarshalInterface(t *testing.T) {
	txt := []byte(`{
  B: first
  A: second
}`)
	var objA testOrderedMapA
	err := Unmarshal(txt, &objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that obj got the elements in the correct order (B before A).
	expectedA := testOrderedMapA{
		NewOrderedMapFromSlice([]KeyValue{
			{"B", "first"},
			{"A", "second"},
		}),
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	buf, err := Marshal(objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that Hjson kept the order from MarshalJSON() (B before A).
	if !bytes.Equal(buf, txt) {
		t.Errorf("Unexpected Hjson output: %s", string(buf))
	}

	var objB testOrderedMapB
	err = Unmarshal(txt, &objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that obj got the elements in the correct order (B before A).
	expectedB := testOrderedMapB{
		NewOrderedMapFromSlice([]KeyValue{
			{"B", "first"},
			{"A", "second"},
		}),
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}

	buf, err = Marshal(objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that Hjson kept the order from MarshalJSON() (B before A).
	if !bytes.Equal(buf, txt) {
		t.Errorf("Unexpected Hjson output: %s", string(buf))
	}
}

func TestUnmarshalInterfaceElemType(t *testing.T) {
	txt := []byte(`{
  B: first
  A: 2
	C: third
}`)
	var objA testOrderedMapA
	err := Unmarshal(txt, &objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedA := testOrderedMapA{
		NewOrderedMapFromSlice([]KeyValue{
			{"B", "first"},
			{"A", "2"},
			{"C", "third"},
		}),
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	objA = testOrderedMapA{}
	pObjA := &objA
	err = Unmarshal(txt, &pObjA)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	var objB testOrderedMapB
	err = Unmarshal(txt, &objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedB := testOrderedMapB{
		NewOrderedMapFromSlice([]KeyValue{
			{"B", "first"},
			{"A", "2"},
			{"C", "third"},
		}),
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}

	objB = testOrderedMapB{}
	pObjB := &objB
	err = Unmarshal(txt, &pObjB)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}
}

func TestUnmarshalSliceMapElemType(t *testing.T) {
	txt := []byte(`[
  {
    B: first
    A: 2
    C: third
  }
  {
    D: 3
  }
]`)
	var objA []testOrderedMapA
	err := Unmarshal(txt, &objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedA := []testOrderedMapA{
		testOrderedMapA{
			NewOrderedMapFromSlice([]KeyValue{
				{"B", "first"},
				{"A", "2"},
				{"C", "third"},
			}),
		},
		testOrderedMapA{
			NewOrderedMapFromSlice([]KeyValue{
				{"D", "3"},
			}),
		},
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	objA = nil
	pObjA := &objA
	err = Unmarshal(txt, &pObjA)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	var objB []testOrderedMapB
	err = Unmarshal(txt, &objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedB := []testOrderedMapB{
		testOrderedMapB{
			NewOrderedMapFromSlice([]KeyValue{
				{"B", "first"},
				{"A", "2"},
				{"C", "third"},
			}),
		},
		testOrderedMapB{
			NewOrderedMapFromSlice([]KeyValue{
				{"D", "3"},
			}),
		},
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}

	objB = nil
	pObjB := &objB
	err = Unmarshal(txt, &pObjB)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}
}

func TestUnmarshalSliceMapPointerElemType(t *testing.T) {
	txt := []byte(`[
  {
    B: first
    A: 2
    C: third
  }
  {
    D: 3
  }
]`)
	var objA []*testOrderedMapA
	err := Unmarshal(txt, &objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedA := []*testOrderedMapA{
		&testOrderedMapA{
			NewOrderedMapFromSlice([]KeyValue{
				{"B", "first"},
				{"A", "2"},
				{"C", "third"},
			}),
		},
		&testOrderedMapA{
			NewOrderedMapFromSlice([]KeyValue{
				{"D", "3"},
			}),
		},
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	objA = nil
	pObjA := &objA
	err = Unmarshal(txt, &pObjA)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	var objB []*testOrderedMapB
	err = Unmarshal(txt, &objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedB := []*testOrderedMapB{
		&testOrderedMapB{
			NewOrderedMapFromSlice([]KeyValue{
				{"B", "first"},
				{"A", "2"},
				{"C", "third"},
			}),
		},
		&testOrderedMapB{
			NewOrderedMapFromSlice([]KeyValue{
				{"D", "3"},
			}),
		},
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}

	objB = nil
	pObjB := &objB
	err = Unmarshal(txt, &pObjB)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}
}

func TestUnmarshalStructElemType(t *testing.T) {
	txt := []byte(`{
  key1: {
    B: first
    A: 2
    C: third
  }
  key2: {
    D: 3
  }
}`)

	type tsA struct {
		Key1 testOrderedMapA
		Key2 testOrderedMapA
	}

	var objA tsA
	err := Unmarshal(txt, &objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedA := tsA{
		Key1: testOrderedMapA{
			NewOrderedMapFromSlice([]KeyValue{
				{"B", "first"},
				{"A", "2"},
				{"C", "third"},
			}),
		},
		Key2: testOrderedMapA{
			NewOrderedMapFromSlice([]KeyValue{
				{"D", "3"},
			}),
		},
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	objA = tsA{}
	pObjA := &objA
	err = Unmarshal(txt, &pObjA)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	type tsB struct {
		Key1 testOrderedMapB
		Key2 testOrderedMapB
	}

	var objB tsB
	err = Unmarshal(txt, &objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedB := tsB{
		Key1: testOrderedMapB{
			NewOrderedMapFromSlice([]KeyValue{
				{"B", "first"},
				{"A", "2"},
				{"C", "third"},
			}),
		},
		Key2: testOrderedMapB{
			NewOrderedMapFromSlice([]KeyValue{
				{"D", "3"},
			}),
		},
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}

	objB = tsB{}
	pObjB := &objB
	err = Unmarshal(txt, &pObjB)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}
}

func TestUnmarshalStructPointerElemType(t *testing.T) {
	txt := []byte(`{
  key1: {
    B: first
    A: 2
    C: third
  }
  key2: {
    D: 3
  }
}`)

	type tsA struct {
		Key1 *testOrderedMapA
		Key2 *testOrderedMapA
	}

	var objA tsA
	err := Unmarshal(txt, &objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedA := tsA{
		Key1: &testOrderedMapA{
			NewOrderedMapFromSlice([]KeyValue{
				{"B", "first"},
				{"A", "2"},
				{"C", "third"},
			}),
		},
		Key2: &testOrderedMapA{
			NewOrderedMapFromSlice([]KeyValue{
				{"D", "3"},
			}),
		},
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	objA = tsA{}
	pObjA := &objA
	err = Unmarshal(txt, &pObjA)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	type tsB struct {
		Key1 *testOrderedMapB
		Key2 *testOrderedMapB
	}

	var objB tsB
	err = Unmarshal(txt, &objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedB := tsB{
		Key1: &testOrderedMapB{
			NewOrderedMapFromSlice([]KeyValue{
				{"B", "first"},
				{"A", "2"},
				{"C", "third"},
			}),
		},
		Key2: &testOrderedMapB{
			NewOrderedMapFromSlice([]KeyValue{
				{"D", "3"},
			}),
		},
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}

	objB = tsB{}
	pObjB := &objB
	err = Unmarshal(txt, &pObjB)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}
}

func TestUnmarshalSliceElemType(t *testing.T) {
	txt := []byte(`[
	1
	two
]`)

	var objA testSliceElemTyperA
	err := Unmarshal(txt, &objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testSliceElemTyper.ElemType()).
	expectedA := testSliceElemTyperA{
		"1",
		"two",
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	objA = nil
	pObjA := &objA
	err = Unmarshal(txt, &pObjA)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	var objB testSliceElemTyperB
	err = Unmarshal(txt, &objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedB := testSliceElemTyperB{
		"1",
		"two",
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}

	objB = nil
	pObjB := &objB
	err = Unmarshal(txt, &pObjB)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}
}

func TestUnmarshalSliceSliceElemType(t *testing.T) {
	txt := []byte(`[
  [
    1
    two
  ]
]`)

	var objA []testSliceElemTyperA
	err := Unmarshal(txt, &objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testSliceElemTyper.ElemType()).
	expectedA := []testSliceElemTyperA{
		testSliceElemTyperA{
			"1",
			"two",
		},
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	objA = nil
	pObjA := &objA
	err = Unmarshal(txt, &pObjA)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	var objB []testSliceElemTyperB
	err = Unmarshal(txt, &objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedB := []testSliceElemTyperB{
		testSliceElemTyperB{
			"1",
			"two",
		},
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}

	objB = nil
	pObjB := &objB
	err = Unmarshal(txt, &pObjB)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}
}

func TestUnmarshalSlicePointerSliceElemType(t *testing.T) {
	txt := []byte(`[
  [
    1
    two
  ]
]`)

	var objA []*testSliceElemTyperA
	err := Unmarshal(txt, &objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testSliceElemTyper.ElemType()).
	expectedA := []*testSliceElemTyperA{
		&testSliceElemTyperA{
			"1",
			"two",
		},
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	objA = nil
	pObjA := &objA
	err = Unmarshal(txt, &pObjA)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	var objB []*testSliceElemTyperB
	err = Unmarshal(txt, &objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedB := []*testSliceElemTyperB{
		&testSliceElemTyperB{
			"1",
			"two",
		},
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}

	objB = nil
	pObjB := &objB
	err = Unmarshal(txt, &pObjB)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}
}

func TestUnmarshalStructSliceElemType(t *testing.T) {
	txt := []byte(`{
  Key1: [
    1
    two
  ]
}`)

	type tsA struct {
		Key1 testSliceElemTyperA
	}

	var objA tsA
	err := Unmarshal(txt, &objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testSliceElemTyper.ElemType()).
	expectedA := tsA{
		Key1: testSliceElemTyperA{
			"1",
			"two",
		},
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	objA = tsA{}
	pObjA := &objA
	err = Unmarshal(txt, &pObjA)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	type tsB struct {
		Key1 testSliceElemTyperB
	}

	var objB tsB
	err = Unmarshal(txt, &objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedB := tsB{
		Key1: testSliceElemTyperB{
			"1",
			"two",
		},
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}

	objB = tsB{}
	pObjB := &objB
	err = Unmarshal(txt, &pObjB)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}
}

func TestUnmarshalStructPointerSliceElemType(t *testing.T) {
	txt := []byte(`{
  Key1: [
    1
    two
  ]
}`)

	type tsA struct {
		Key1 *testSliceElemTyperA
	}

	var objA tsA
	err := Unmarshal(txt, &objA)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testSliceElemTyper.ElemType()).
	expectedA := tsA{
		Key1: &testSliceElemTyperA{
			"1",
			"two",
		},
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	objA = tsA{}
	pObjA := &objA
	err = Unmarshal(txt, &pObjA)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objA, expectedA) {
		t.Errorf("Unexpected obj: %#v\n", objA)
	}

	type tsB struct {
		Key1 *testSliceElemTyperB
	}

	var objB tsB
	err = Unmarshal(txt, &objB)
	if err != nil {
		t.Error(err)
	}
	// Make sure that all values are strings (because of testOrderedMap.ElemType()).
	expectedB := tsB{
		Key1: &testSliceElemTyperB{
			"1",
			"two",
		},
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}

	objB = tsB{}
	pObjB := &objB
	err = Unmarshal(txt, &pObjB)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(objB, expectedB) {
		t.Errorf("Unexpected obj: %#v\n", objB)
	}
}

func TestJSONNumber(t *testing.T) {
	var v interface{}
	b := []byte("35e-7")
	err := UnmarshalWithOptions(b, &v, DecoderOptions{UseJSONNumber: true})
	if err != nil {
		t.Error(err)
	}
	if v.(json.Number).String() != string(b) {
		t.Errorf("Expected %s, got %v\n", string(b), v)
	}

	b2, err := Marshal(v)
	if err != nil {
		t.Error(err)
	}
	if string(b2) != string(b) {
		t.Errorf("Expected %s, got %v\n", string(b), string(b2))
	}

	var n json.Number
	err = Unmarshal(b, &n)
	if err != nil {
		t.Error(err)
	}
	if n.String() != string(b) {
		t.Errorf("Expected %s, got %v\n", string(b), n)
	}
	f, err := n.Float64()
	if err != nil {
		t.Error(err)
	}
	if math.Abs(f-35e-7) > 1e-7 {
		t.Errorf("Expected %f, got %f\n", 35e-7, f)
	}
	_, err = n.Int64()
	if err == nil {
		t.Errorf("Did not expect %v to be parsable to int64", n)
	}
}

func TestMapKeys(t *testing.T) {
	sampleText := []byte(`
4: four
3: true
2: 2
1: null
`)

	{
		var v map[string]interface{}
		err := Unmarshal(sampleText, &v)
		if err != nil {
			t.Error(err)
		} else {
			if v["3"] != true {
				t.Errorf("Expected boolean, got %v", reflect.TypeOf(v["3"]))
			}
			if v["2"] != 2.0 {
				t.Errorf("Expected float64, got %v", reflect.TypeOf(v["2"]))
			}
			if v["1"] != nil {
				t.Errorf("Expected nil-interface, got %v", reflect.TypeOf(v["1"]))
			}
		}
	}

	{
		var v map[int]interface{}
		err := Unmarshal(sampleText, &v)
		if err != nil {
			t.Error(err)
		} else {
			if v[3] != true {
				t.Errorf("Expected boolean, got %v", reflect.TypeOf(v[3]))
			}
			if v[2] != 2.0 {
				t.Errorf("Expected float64, got %v", reflect.TypeOf(v[2]))
			}
			if v[1] != nil {
				t.Errorf("Expected nil-interface, got %v", reflect.TypeOf(v[1]))
			}
		}
	}

	{
		var v map[string]string
		err := Unmarshal(sampleText, &v)
		if err != nil {
			t.Error(err)
		} else {
			if v["3"] != "true" {
				t.Errorf("Expected true, got %v", v["3"])
			}
			if v["2"] != "2" {
				t.Errorf("Expected 2, got %v", v["2"])
			}
			if v["1"] != "null" {
				t.Errorf("Expected null, got %v", v["1"])
			}
		}
	}

	{
		var v map[int]string
		err := Unmarshal(sampleText, &v)
		if err != nil {
			t.Error(err)
		} else {
			if v[3] != "true" {
				t.Errorf("Expected true, got %v", v[3])
			}
			if v[2] != "2" {
				t.Errorf("Expected 2, got %v", v[2])
			}
			if v[1] != "null" {
				t.Errorf("Expected null, got %v", v[1])
			}
		}
	}
}

func TestMapTree(t *testing.T) {
	textA := []byte(`
4: four
3: true
5: {
  sub1: 1
	sub2: 2
}
2: 2
1: null
`)

	textB := []byte(`
4: five
5: {
	sub2: 3
}
`)

	var v map[int]interface{}
	err := Unmarshal(textA, &v)
	if err != nil {
		t.Error(err)
	}

	err = Unmarshal(textB, &v)
	if err != nil {
		t.Error(err)
	} else {
		// Note that the map on key 5 was fully replaced by textB.
		if !reflect.DeepEqual(v, map[int]interface{}{
			1: nil,
			2: 2.0,
			3: true,
			4: "five",
			5: map[string]interface{}{
				"sub2": 3.0,
			},
		}) {
			t.Errorf("Unexpected map values:\n%#v\n", v)
		}
	}
}

func TestStructTree(t *testing.T) {
	type tsB struct {
		Sub1 string
		Sub2 string
	}

	type tsA struct {
		One   *int
		Two   int
		Three bool
		Four  string
		Five  tsB
	}

	textA := []byte(`
four: four
three: true
five: {
  sub1: 1
	sub2: 2
}
two: 2
one: null
`)

	textB := []byte(`
four: five
five: {
	sub2: 3
}
`)

	var v tsA
	err := Unmarshal(textA, &v)
	if err != nil {
		t.Error(err)
	}

	err = Unmarshal(textB, &v)
	if err != nil {
		t.Error(err)
	} else {
		// Note that only the field Sub2 was replaced by textB in the tsB struct.
		// The field Sub1 still has the value that was set by textA.
		if !reflect.DeepEqual(v, tsA{
			One:   nil,
			Two:   2,
			Three: true,
			Four:  "five",
			Five: tsB{
				Sub1: "1",
				Sub2: "3",
			},
		}) {
			t.Errorf("Unexpected struct values:\n%#v\n", v)
		}
	}
}

type itsF struct {
	itsG
	F string
}

type itsG struct {
	*itsH
	G string
}

type itsH struct {
	itsI
	H string
}

type itsI struct {
	I string
}

func TestEmbeddedStructTree(t *testing.T) {
	textA := []byte(`
f: 1.5
g: true
h: null
i: false
`)

	sA := itsF{
		itsG: itsG{
			itsH: &itsH{},
		},
	}
	err := Unmarshal(textA, &sA)
	if err != nil {
		t.Error(err)
	} else {
		buf, err := json.MarshalIndent(sA, "", "  ")
		if err != nil {
			t.Error(err)
		}
		// Note that only the field Sub2 was replaced by textB in the tsB struct.
		// The field Sub1 still has the value that was set by textA.
		if !reflect.DeepEqual(sA, itsF{
			F: "1.5",
			itsG: itsG{
				G: "true",
				itsH: &itsH{
					H: "null",
					itsI: itsI{
						I: "false",
					},
				},
			},
		}) {
			t.Errorf("Unexpected struct values:\n%v\n", string(buf))
		}
	}
}

type InterfaceA interface {
	FuncA() string
}

type itsB struct {
	Sub1 string
	Sub2 string
}

type itsA struct {
	One   *int
	Two   int
	Three bool
	Four  *string
	Five  InterfaceA
}

func (c *itsB) FuncA() string {
	return c.Sub1
}

func (c *itsA) FuncA() string {
	return *c.Four
}

func TestStructInterface(t *testing.T) {
	textA := []byte(`
four: 4
three: true
five: {
  sub1: 1
	sub2: 2
}
two: 2
one: null
`)

	textB := []byte(`
four: 5
five: {
	sub2: 3
}
`)

	sA := itsA{
		Five: &itsB{},
	}
	err := Unmarshal(textA, &sA)
	if err != nil {
		t.Error(err)
	}

	err = Unmarshal(textB, &sA)
	if err != nil {
		t.Error(err)
	} else {
		five := "5"
		// Note that only the field Sub2 was replaced by textB in the tsB struct.
		// The field Sub1 still has the value that was set by textA.
		if !reflect.DeepEqual(sA, itsA{
			One:   nil,
			Two:   2,
			Three: true,
			Four:  &five,
			Five: &itsB{
				Sub1: "1",
				Sub2: "3",
			},
		}) {
			t.Errorf("Unexpected struct values:\n%#v\n", sA)
		}
	}
}

type itsC string

func (c itsC) FuncA() string {
	return string(c)
}

func TestStringInterface(t *testing.T) {
	textA := []byte(`3`)

	var sA itsC
	var isA InterfaceA
	isA = &sA
	err := Unmarshal(textA, &isA)
	if err != nil {
		t.Error(err)
	} else {
		if string(sA) != "3" {
			t.Errorf("Unexpected string value:\n%v\n", sA)
		}
	}
}

func TestStringPointer(t *testing.T) {
	textA := []byte(`3`)

	var psA *itsC
	err := Unmarshal(textA, &psA)
	if err != nil {
		t.Error(err)
	} else {
		if string(*psA) != "3" {
			t.Errorf("Unexpected string value:\n%v\n", *psA)
		}
	}
}

type itsD []*itsC

func (c itsD) FuncA() string {
	return ""
}

func TestSliceInterface(t *testing.T) {
	textA := []byte(`
[
	3
	alfa
	5
]
`)

	var sA itsD
	var isA InterfaceA
	isA = &sA
	err := Unmarshal(textA, &isA)
	if err != nil {
		t.Error(err)
	} else {
		a := itsC("3")
		b := itsC("alfa")
		c := itsC("5")
		if !reflect.DeepEqual(sA, itsD{
			&a,
			&b,
			&c,
		}) {
			buf, _ := json.Marshal(sA)
			t.Errorf("Unexpected slice values:\n%v\n", string(buf))
		}
	}
}

type InterfaceB interface{}

func TestNilInterfaces(t *testing.T) {
	textA := []byte(`
[
	3
	alfa
	5
]
`)

	textB := []byte(`
four: five
five: {
	sub2: 3
}
`)

	var isA InterfaceA
	err := Unmarshal(textA, &isA)
	if err == nil {
		// If the interface has at least one function it must not be empty.
		t.Error("Unmarshal into empty InterfaceA did not return error")
	}

	var isB InterfaceB
	err = Unmarshal(textA, &isB)
	if err != nil {
		t.Error(err)
	}

	var isC interface{}
	err = Unmarshal(textA, &isC)
	if err != nil {
		t.Error(err)
	}

	var isD itsA
	err = Unmarshal(textB, &isD)
	if err == nil {
		// If the interface has at least one function it must not be empty.
		t.Error("Unmarshal into empty InterfaceA did not return error")
	}
}

type itsE struct {
	One string
	Two *itsB
}

func TestStructPointer(t *testing.T) {
	textA := []byte(`
one: 1
two: {
	sub2: 3
}
`)

	var psA *itsE
	err := Unmarshal(textA, &psA)
	if err != nil {
		t.Error(err)
	} else {
		if !reflect.DeepEqual(psA, &itsE{
			One: "1",
			Two: &itsB{
				Sub2: "3",
			},
		}) {
			buf, _ := json.Marshal(psA)
			t.Errorf("Unexpected struct values:\n%v\n", string(buf))
		}
	}
}

func TestFieldCase(t *testing.T) {
	type tsA struct {
		Field int
		FiEld int
		FieLd string
		FielD int
	}

	textA := []byte(`
FieLd: 3
`)

	var sA tsA
	err := Unmarshal(textA, &sA)
	if err != nil {
		t.Error(err)
	} else if sA.FieLd != "3" {
		t.Errorf("Unexpected struct values:\n%#v\n", sA)
	}
}

type itsJ struct {
	anon string
}

func (c *itsJ) UnmarshalText(text []byte) error {
	c.anon = string(text)
	return nil
}

func (c itsJ) MarshalJSON() ([]byte, error) {
	return []byte("{\"anon\": \"" + strings.Replace(c.anon, "\"", "\\\"", -1) + "\"}"), nil
}

type itsK *itsJ

type itsL int

func (c *itsL) UnmarshalText(text []byte) error {
	if len(text) > 0 {
		*c = itsL(text[0])
	}
	return nil
}

func (c itsL) MarshalJSON() ([]byte, error) {
	return []byte(string(rune(c))), nil
}

type itsM itsL

func TestUnmarshalText(t *testing.T) {
	type tsA struct {
		A       itsJ
		B       *itsJ
		C       encoding.TextUnmarshaler
		D       itsL
		E       itsM // Does not implement encoding.TextUnmarshaler, will receive int.
		F       *itsM
		G       *itsJ
		H       *itsJ
		I       *itsJ
		J       *string
		K       string
		L       string
		Novalue *string
	}

	textA := []byte(`
a: 3
b: 4
c: 5
d: 6
e: 7
f: null, g: "a text"
h: null, i: "second text"
j: null, k: "another text", l: null
`)
	sA := tsA{
		B: &itsJ{},
		C: &itsJ{},
	}
	err := Unmarshal(textA, &sA)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(sA, tsA{
		A:       itsJ{"3"},
		B:       &itsJ{"4"},
		C:       &itsJ{"5"},
		D:       itsL([]byte(`6`)[0]),
		E:       7,
		F:       nil,
		G:       &itsJ{"a text"},
		H:       nil,
		I:       &itsJ{"second text"},
		J:       nil,
		K:       "another text",
		L:       "null",
		Novalue: nil,
	}) {
		out, errJ := json.MarshalIndent(sA, "", "  ")
		if errJ != nil {
			t.Error(errJ)
		}
		t.Errorf("Unexpected struct values:\n%s\n", out)
	}

	textB := []byte(`8`)

	var sL itsL
	err = Unmarshal(textB, &sL)
	if err != nil {
		t.Error(err)
	} else if sL != itsL(textB[0]) {
		t.Errorf("Unexpected sL value: %v\n", sL)
	}

	var m map[string]itsL
	err = Unmarshal(textA, &m)
	if err == nil {
		t.Error("Should have failed, should not be possible to call pointer method UnmarshalText() on the map elements because they are not addressable.")
	}
}

type UnmarshalExtraFieldsConfig struct {
	Type  string `json:"type"`
	Extra map[string]interface{}
}

// We implement UnmarshalJSON to capture the extra fields
func (c *UnmarshalExtraFieldsConfig) UnmarshalJSON(data []byte) error {
	type Alias UnmarshalExtraFieldsConfig
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	// This will unmarshal the known fields
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Now unmarshal everything into a map to get extra fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if val, ok := raw["allow_local"]; ok {
		if _, isBool := val.(bool); !isBool {
			return fmt.Errorf("allow_local is not bool, got %T", val)
		}
	} else {
		return fmt.Errorf("allow_local missing")
	}
	return nil
}

func TestUnmarshalStructWithExtraFields(t *testing.T) {
	txt := `
{
  type: network_allow
  allow_local: true
}
`
	var cfg UnmarshalExtraFieldsConfig
	err := Unmarshal([]byte(txt), &cfg)
	if err != nil {
		if strings.Contains(err.Error(), "allow_local is not bool") {
			t.Fatalf("Bug reproduced: %v", err)
		}
		t.Fatalf("Unexpected error: %v", err)
	}
}
