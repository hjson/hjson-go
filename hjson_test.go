package hjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func getContent(file string) []byte {
	if data, err := ioutil.ReadFile(file); err != nil {
		panic(err)
	} else {
		// The output from Marshal() always uses Unix EOL, but git might have
		// converted files to Windows EOL on Windows, therefore we convert all
		// "\r\n" to "\n".
		return bytes.Replace(data, []byte("\r\n"), []byte("\n"), -1)
	}
}

func getTestContent(name string) []byte {
	p := fmt.Sprintf("./assets/%s_test.hjson", name)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		p = fmt.Sprintf("./assets/%s_test.json", name)
	}
	return getContent(p)
}

func getResultContent(name string) ([]byte, []byte) {
	p1 := fmt.Sprintf("./assets/sorted/%s_result.json", name)
	p2 := fmt.Sprintf("./assets/sorted/%s_result.hjson", name)
	return getContent(p1), getContent(p2)
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
			panic(err)
		} else {
			return
		}
	} else if shouldFail {
		panic(errors.New(name + " should_fail!"))
	}

	rjson, rhjson := getResultContent(name)

	actualHjson, _ := Marshal(data)
	actualHjson = append(actualHjson, '\n')
	actualJSON, _ := json.MarshalIndent(data, "", "  ")
	actualJSON = fixJSON(actualJSON)

	// add fixes where go's json differs from javascript
	switch name {
	case "kan":
		actualJSON = []byte(strings.Replace(string(actualJSON), "    -0,", "    0,", -1))
	case "pass1":
		actualJSON = []byte(strings.Replace(string(actualJSON), "1.23456789e+09", "1234567890", -1))
	}

	hjsonOK := bytes.Equal(rhjson, actualHjson)
	jsonOK := bytes.Equal(rjson, actualJSON)
	if !hjsonOK {
		t.Logf("%s\n---hjson expected\n%s\n---hjson actual\n%s\n---\n", name, rhjson, actualHjson)
	}
	if !jsonOK {
		t.Logf("%s\n---json expected\n%s\n---json actual\n%s\n---\n", name, rjson, actualJSON)
	}
	if !hjsonOK || !jsonOK {
		panic("fail!")
	}
}

func TestHjson(t *testing.T) {

	files := strings.Split(string(getContent("assets/testlist.txt")), "\n")

	for _, file := range files {
		if !strings.HasPrefix(file, "stringify/quotes") && !strings.HasPrefix(file, "extra/") {
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

type MyUnmarshaller struct {
	A string
	x string
}

func (c *MyUnmarshaller) UnmarshalJSON(in []byte) error {
	var out map[string]interface{}
	err := Unmarshal(in, &out)
	if err != nil {
		return err
	}
	a, ok := out["A"]
	if !ok {
		return errors.New("Missing key")
	}
	b, ok := a.(string)
	if !ok {
		return errors.New("Not a string")
	}
	c.x = b
	return nil
}

func TestUnmarshalInterface(t *testing.T) {
	var obj MyUnmarshaller
	err := Unmarshal([]byte("A: test"), &obj)
	if err != nil {
		t.Error(err)
	}
	if obj.A != "" || obj.x != "test" {
		t.Errorf("Unexpected obj values: %+v", obj)
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

func TestUnmarshalPartially(t *testing.T) {
	var src = `
	{
		database:
		{
		  host: 127.0.0.1
		  port: 555
		}
	  }
	@@there is sth cannot be parsed as hijson.
	`
	v, next, err := UnmarshalPartially([]byte(src))
	if err != nil {
		panic(err)
	}
	if next != strings.Index(src, "@") {
		panic("wrong next value")
	}
	if v == nil {
		panic("v is nil")
	}
	val, ok := v.(map[string]interface{})
	if !ok {
		panic("v has wrong type")
	}
	_, ok = val["database"]
	if !ok {
		panic("v has wrong value")
	}
}
