package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime/debug"

	"github.com/hjson/hjson-go/v4"
)

// Can be set when building for example like this:
// go build -ldflags "-X main.Version=v3.0"
var Version string

func fixJSON(data []byte) []byte {
	data = bytes.Replace(data, []byte("\\u003c"), []byte("<"), -1)
	data = bytes.Replace(data, []byte("\\u003e"), []byte(">"), -1)
	data = bytes.Replace(data, []byte("\\u0026"), []byte("&"), -1)
	data = bytes.Replace(data, []byte("\\u0008"), []byte("\\b"), -1)
	data = bytes.Replace(data, []byte("\\u000c"), []byte("\\f"), -1)
	return data
}

func main() {

	flag.Usage = func() {
		fmt.Println("usage: hjson-cli [OPTIONS] [INPUT]")
		fmt.Println("hjson can be used to convert JSON from/to Hjson.")
		fmt.Println("")
		fmt.Println("hjson will read the given JSON/Hjson input file or read from stdin.")
		fmt.Println("")
		fmt.Println("Options:")
		flag.PrintDefaults()
	}

	var help = flag.Bool("h", false, "Show this screen.")
	var showJSON = flag.Bool("j", false, "Output as formatted JSON.")
	var showCompact = flag.Bool("c", false, "Output as JSON.")

	var indentBy = flag.String("indentBy", "  ", "The indent string.")
	var bracesSameLine = flag.Bool("bracesSameLine", false, "Print braces on the same line.")
	var omitRootBraces = flag.Bool("omitRootBraces", false, "Omit braces at the root.")
	var quoteAlways = flag.Bool("quoteAlways", false, "Always quote string values.")
	var showVersion = flag.Bool("v", false, "Show version.")
	var preserveKeyOrder = flag.Bool("preserveKeyOrder", false, "Preserve key order in objects/maps.")
	var preserveComments = flag.Bool("preserveComments", false, "Preserve comments in Hjson output (and key order in any output).")

	flag.Parse()
	if *help || flag.NArg() > 1 {
		flag.Usage()
		os.Exit(1)
	}

	if *showVersion {
		if Version != "" {
			fmt.Println(Version)
		} else if bi, ok := debug.ReadBuildInfo(); ok {
			fmt.Println(bi.Main.Version)
		} else {
			fmt.Println("Unknown version")
		}
		os.Exit(0)
	}

	var err error
	var data []byte
	if flag.NArg() == 1 {
		data, err = ioutil.ReadFile(flag.Arg(0))
	} else {
		data, err = ioutil.ReadAll(os.Stdin)
	}
	if err != nil {
		panic(err)
	}

	var value interface{}

	if *preserveKeyOrder || *preserveComments {
		var node *hjson.Node
		opt := hjson.DefaultDecoderOptions()
		opt.WhitespaceAsComments = false
		err = hjson.UnmarshalWithOptions(data, &node, opt)
		value = node
	} else {
		err = hjson.Unmarshal(data, &value)
	}
	if err != nil {
		panic(err)
	}

	var out []byte
	if *showCompact {
		out, err = json.Marshal(value)
		if err != nil {
			panic(err)
		}
		out = fixJSON(out)
	} else if *showJSON {
		out, err = json.MarshalIndent(value, "", *indentBy)
		if err != nil {
			panic(err)
		}
		out = fixJSON(out)
	} else {
		opt := hjson.DefaultOptions()
		opt.IndentBy = *indentBy
		opt.BracesSameLine = *bracesSameLine
		opt.EmitRootBraces = !*omitRootBraces
		opt.QuoteAlways = *quoteAlways
		opt.Comments = *preserveComments
		out, err = hjson.MarshalWithOptions(value, opt)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println(string(out))
}
