package hjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
	var dat map[string]interface{}
	err := Unmarshal([]byte(`[1,2,3,4]`), &dat)
	if err == nil {
		t.Errorf("Should have failed when trying to unmarshal an array to a map.")
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
