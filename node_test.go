package hjson

import (
	"testing"
)

func verifyNodeContent(t *testing.T, node *Node, txtExpected string) {
	opt := DefaultOptions()
	opt.EmitRootBraces = false
	bOut, err := MarshalWithOptions(node, opt)
	if err != nil {
		t.Error(err)
	}

	if string(bOut) != txtExpected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s\n\n", txtExpected, string(bOut))
	}
}

func TestNode1(t *testing.T) {
	txt := `b: 1
a: 2`

	var node *Node
	Unmarshal([]byte(txt), &node)

	verifyNodeContent(t, node, txt)
}
