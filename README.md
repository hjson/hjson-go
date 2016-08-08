# hjson-go

[![Build Status](https://img.shields.io/travis/laktak/hjson-go.svg?style=flat-square)](http://travis-ci.org/laktak/hjson-go)
[![Releases](https://img.shields.io/github/release/laktak/hjson-go.svg?style=flat-square)](https://github.com/laktak/hjson-go/releases)

![Hjson Intro](http://hjson.org/hjson1.gif)

```
{
  # specify rate in requests/second (because comments are helpful!)
  rate: 1000

  // prefer c-style comments?
  /* feeling old fashioned? */

  # did you notice that rate doesn't need quotes?
  hey: look ma, no quotes for strings either!

  # best of all
  notice: []
  anything: ?

  # yes, commas are optional!
}
```

The Go implementation of Hjson is based on [hjson-js](https://github.com/laktak/hjson-js). For other platforms see [hjson.org](http://hjson.org).

# Install

Make sure you have a working Go environment. See the [install instructions](http://golang.org/doc/install.html).

To install hjson, simply run:

`go get -u github.com/laktak/hjson-go`

## From the Commandline

Install with `go get -u github.com/laktak/hjson-go/hjson-cli`

```
usage: hjson-cli [OPTIONS] [INPUT]
hjson can be used to convert JSON from/to Hjson.

hjson will read the given JSON/Hjson input file or read from stdin.

Options:
  -allowMinusZero
      Allow -0.
  -bracesSameLine
      Print braces on the same line.
  -c  Output as JSON.
  -h  Show this screen.
  -indentBy string
      The indent string. (default "  ")
  -j  Output as formatted JSON.
  -omitRootBraces
      Omit braces at the root.
  -quoteAlways
      Always quote string values.
```

Sample:
- run `hjson-cli test.json > test.hjson` to convert to Hjson
- run `hjson-cli -j test.hjson > test.json` to convert to JSON

# Usage

```go

package main

import (
  "github.com/laktak/hjson-go"
  "fmt"
)

func main() {

    // Now let's look at decoding JSON data into Go
    // values.
    sample := []byte(`{"num":6.13,"strs":["a","b"]}`)

    // We need to provide a variable where Hjson
    // can put the decoded data.
    var dat map[string]interface{}

    // Decode and a check for errors.
    if err := hjson.Unmarshal(sample, &dat); err != nil {
        panic(err)
    }
    fmt.Println(dat)

    // In order to use the values in the decoded map,
    // we'll need to cast them to their appropriate type.

    num := dat["num"].(float64)
    fmt.Println(num)

    strs := dat["strs"].([]interface{})
    str1 := strs[0].(string)
    fmt.Println(str1)


    // To encode to Hjson with default options:
    sampleMap := map[string]int{"apple": 5, "lettuce": 7}
    hjson, _ := hjson.Marshal(sampleMap)
    // this is short for:
    // options := hjson.DefaultOptions()
    // hjson, _ := hjson.MarshalWithOptions(sampleMap, options)
    fmt.Println(string(hjson))
}

```
