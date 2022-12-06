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
	Unmarshal([]byte(txt), &node)

	verifyNodeContent(t, node, txt)
}

func TestNode2(t *testing.T) {
	txt := `# comment before
b: 1  # comment after
// Comment B4
a: 2
/* Last comment */`

	var node *Node
	Unmarshal([]byte(txt), &node)

	if node.Len() != 2 {
		t.Errorf("Unexpected map length: %v", node.Len())
	}

	aVal, ok, err := node.GetKey("a")
	if err != nil {
		t.Error(err)
	}
	// The value will be a float64 even though it was written without decimals.
	if aVal != 2.0 {
		t.Errorf("Unexpected value for key 'a': %v", aVal)
	} else if !ok {
		t.Errorf("node.GetKey('a') returned false")
	}

	bVal, err := node.GetIndex(0)
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

	firstVal, err := node.GetIndex(0)
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

	node.SetIndex(1, "abcdef")

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
	Unmarshal([]byte(txt), &node)

	verifyNodeContent(t, node, txt)
}
