

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"hjson"
	"io/ioutil"
	"os"
)

func fixJson(data []byte) ([]byte) {
	data = bytes.Replace(data, []byte("\\u003c"), []byte("<"), -1)
	data = bytes.Replace(data, []byte("\\u003e"), []byte(">"), -1)
	data = bytes.Replace(data, []byte("\\u0026"), []byte("&"), -1)
	return data
}

func main() {

	flag.Usage = func() {
		fmt.Println("usage: hjson [OPTIONS] [INPUT]")
		fmt.Println("hjson can be used to convert JSON from/to Hjson.")
		fmt.Println("")
		fmt.Println("hjson will read the given JSON/Hjson input file or read from stdin.")
		fmt.Println("")
		fmt.Println("Options:")
		flag.PrintDefaults()
	}

	var help = flag.Bool("h", false, "Show this screen.")
	var showJson = flag.Bool("j", false, "Output as formatted JSON.")
	var showCompact = flag.Bool("c", false, "Output as JSON.")
	// var showVersion = flag.Bool("V", false, "Show version.")

	flag.Parse()
	if *help || flag.NArg() > 1 {
		fmt.Println("{}", flag.NArg())
		flag.Usage()
		os.Exit(1)
	}

	var err error
	var data []byte
	if flag.NArg() == 1 {
		data, err = ioutil.ReadFile(flag.Arg(0))
	} else {
		data, err = ioutil.ReadAll(os.Stdin)
	}
	if err != nil { panic(err) }

	// fmt.Print(string(data))

	var value interface{}

	if err := hjson.Unmarshal(data, &value); err != nil {
		panic(err)
	}

	var out []byte
	if *showCompact {
		out, _ = json.Marshal(value)
		out = fixJson(out)
	} else if *showJson {
		out, _ = json.MarshalIndent(value, "", "  ")
		out = fixJson(out)
	} else {
		out, _ = json.MarshalIndent(value, "", "  ")
		out = fixJson(out)
		//out, _ = hjson.MarshalIndent(value, "", "	")
	}

	fmt.Println(string(out))
}
