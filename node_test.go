package hjson

import (
	"encoding/json"
	"testing"
)

func compareStrings(t *testing.T, bOut []byte, txtExpected string) {
	if string(bOut) != txtExpected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s\n\n", txtExpected, string(bOut))
	}
}

func verifyNodeContent(t *testing.T, node *Node, txtExpected string) {
	opt := DefaultOptions()
	opt.EmitRootBraces = false
	bOut, err := MarshalWithOptions(node, opt)
	if err != nil {
		t.Error(err)
	}

	compareStrings(t, bOut, txtExpected)
}

func TestNode1(t *testing.T) {
	txt := `b: 1
a: 2`

	var node *Node
	err := Unmarshal([]byte(txt), &node)
	if err != nil {
		t.Error(err)
	}

	verifyNodeContent(t, node, txt)
}

func TestNode2(t *testing.T) {
	txt := `# comment before
b: 1  # comment after
// Comment B4
a: 2
/* Last comment */`

	var node *Node
	err := Unmarshal([]byte(txt), &node)
	if err != nil {
		t.Error(err)
	}

	if node.Len() != 2 {
		t.Errorf("Unexpected map length: %v", node.Len())
	}

	aVal, ok, err := node.AtKey("a")
	if err != nil {
		t.Error(err)
	}
	// The value will be a float64 even though it was written without decimals.
	if aVal != 2.0 {
		t.Errorf("Unexpected value for key 'a': %v", aVal)
	} else if !ok {
		t.Errorf("node.AtKey('a') returned false")
	}

	bVal, err := node.AtIndex(0)
	if err != nil {
		t.Error(err)
	}
	// The value will be a float64 even though it was written without decimals.
	if bVal != 1.0 {
		t.Errorf("Unexpected value for key 'b': %v", bVal)
	}

	verifyNodeContent(t, node, txt)

	opt := DefaultOptions()
	opt.Comments = false
	bOut, err := MarshalWithOptions(node, opt)
	if err != nil {
		t.Error(err)
	}

	compareStrings(t, bOut, `{
  b: 1
  a: 2
}`)

	bOut, err = json.Marshal(node)
	if err != nil {
		t.Error(err)
	}

	compareStrings(t, bOut, `{"b":1,"a":2}`)

	intLen := node.Value.(*OrderedMap).Map["b"].(*Node).Len()
	if intLen != 0 {
		t.Errorf("Unexpected int length: %v", intLen)
	}

	node.SetIndex(0, 3)

	verifyNodeContent(t, node, `# comment before
b: 3  # comment after
// Comment B4
a: 2
/* Last comment */`)

	node.SetKey("b", "abcdef")

	verifyNodeContent(t, node, `# comment before
b: "abcdef"  # comment after
// Comment B4
a: 2
/* Last comment */`)

	strLen := node.Value.(*OrderedMap).Map["b"].(*Node).Len()
	if strLen != 6 {
		t.Errorf("Unexpected string length: %v", strLen)
	}

	node.Value.(*OrderedMap).Map["b"] = "xyz"

	verifyNodeContent(t, node, `b: xyz
// Comment B4
a: 2
/* Last comment */`)
}

func TestNode3(t *testing.T) {
	txt := `# comment before
[
# after [
  1  # comment after
  // Comment B4
  2
    # COmment After
]
/* Last comment */`

	var node *Node
	Unmarshal([]byte(txt), &node)

	if node.Len() != 2 {
		t.Errorf("Unexpected slice length: %v", node.Len())
	}

	firstVal, err := node.AtIndex(0)
	if err != nil {
		t.Error(err)
	}
	// The value will be a float64 even though it was written without decimals.
	if firstVal != 1.0 {
		t.Errorf("Unexpected value for index 0: %v", firstVal)
	}

	verifyNodeContent(t, node, txt)

	opt := DefaultOptions()
	opt.Comments = false
	bOut, err := MarshalWithOptions(node, opt)
	if err != nil {
		t.Error(err)
	}

	compareStrings(t, bOut, `[
  1
  2
]`)

	bOut, err = json.Marshal(node)

	compareStrings(t, bOut, `[1,2]`)

	intLen := node.Value.([]interface{})[1].(*Node).Len()
	if intLen != 0 {
		t.Errorf("Unexpected int length: %v", intLen)
	}

	err = node.SetIndex(1, "abcdef")
	if err != nil {
		t.Error(err)
	}

	verifyNodeContent(t, node, `# comment before
[
# after [
  1  # comment after
  // Comment B4
  abcdef
    # COmment After
]
/* Last comment */`)

	strLen := node.Value.([]interface{})[1].(*Node).Len()
	if strLen != 6 {
		t.Errorf("Unexpected string length: %v", strLen)
	}

	node.Value.([]interface{})[0] = "xyz"

	verifyNodeContent(t, node, `# comment before
[
  xyz
  // Comment B4
  abcdef
    # COmment After
]
/* Last comment */`)
}

func TestNode4(t *testing.T) {
	txt := `# comment before
b: /* key comment */ {
  sub1: 1  # comment after
} # cm after obj
// Comment B4
a: 2
/* Last comment */`

	var node *Node
	err := Unmarshal([]byte(txt), &node)
	if err != nil {
		t.Error(err)
	}

	verifyNodeContent(t, node, txt)

	sub1Val, ok, err := node.NK("b").AtKey("sub1")
	if err != nil {
		t.Error(err)
	}
	// The value will be a float64 even though it was written without decimals.
	if sub1Val != 1.0 {
		t.Errorf("Unexpected value for sub1: %v", sub1Val)
	} else if !ok {
		t.Errorf("AtKey('sub1') returned false")
	}

	sub1Val, ok, err = node.NK("Z").AtKey("sub2")
	if err != nil {
		t.Error(err)
	}
	if ok {
		t.Errorf("Should have returned false when calling AtKey() on nil")
	}

	err = node.NK("Z").SetKey("sub2", 3)
	if err == nil {
		t.Errorf("Should have returned an error calling SetKey() on nil")
	}

	err = node.NKC("Z").SetKey("sub2", 3)
	if err != nil {
		t.Error(err)
	}

	err = node.NKC("Z").SetKey("sub2", 4)
	if err != nil {
		t.Error(err)
	}

	err = node.NKC("X").NKC("Y").SetKey("sub3", 5)
	if err != nil {
		t.Error(err)
	}

	verifyNodeContent(t, node, `# comment before
b: /* key comment */ {
  sub1: 1  # comment after
} # cm after obj
// Comment B4
a: 2
/* Last comment */
Z: {
  sub2: 4
}
X: {
  Y: {
    sub3: 5
  }
}`)
}

func TestDisallowDuplicateKeys(t *testing.T) {
	txt := `a: 1
a: 2
b: 3
c: 4
b: 5`

	var node *Node
	err := Unmarshal([]byte(txt), &node)
	if err != nil {
		t.Error(err)
	}

	verifyNodeContent(t, node, `a: 2
b: 5
c: 4`)

	decOpt := DefaultDecoderOptions()
	decOpt.DisallowDuplicateKeys = true
	err = UnmarshalWithOptions([]byte(txt), &node, decOpt)
	if err == nil {
		t.Errorf("Should have returned error because of duplicate keys.")
	}
}

func TestWhitespaceAsComments(t *testing.T) {
	txt := `

a: 2
   b: 3

`

	var node *Node
	err := Unmarshal([]byte(txt), &node)
	if err != nil {
		t.Error(err)
	}

	verifyNodeContent(t, node, txt)

	decOpt := DefaultDecoderOptions()
	decOpt.WhitespaceAsComments = false
	err = UnmarshalWithOptions([]byte(txt), &node, decOpt)
	if err != nil {
		t.Error(err)
	}

	verifyNodeContent(t, node, `a: 2
b: 3`)
}

func TestDeclareNode(t *testing.T) {
	var node Node

	node2 := node.NK("a")
	if node2 != nil {
		t.Errorf("node.NK() created a node")
	}

	err := node.NKC("a").NKC("aa").NKC("aaa").SetKey("aaaa", "a string")
	if err != nil {
		t.Error(err)
	}
	err = node.SetKey("b", 2)
	if err != nil {
		t.Error(err)
	}
	err = node.SetIndex(1, 3.0)
	if err != nil {
		t.Error(err)
	}

	verifyNodeContent(t, &node, `a: {
  aa: {
    aaa: {
      aaaa: a string
    }
  }
}
b: 3`)
}
