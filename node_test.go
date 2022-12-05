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

	node.Value.(*OrderedMap).Map["b"].(*Node).Value = 3

	verifyNodeContent(t, node, `# comment before
b: 3  # comment after
// Comment B4
a: 2
/* Last comment */`)
}
