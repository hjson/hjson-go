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

	bKey, bVal, err := node.AtIndex(0)
	if err != nil {
		t.Error(err)
	}
	if bKey != "b" {
		t.Errorf("Expected key 'b', got: %v", bKey)
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

	oldVal, found, err := node.SetKey("b", "abcdef")
	if err != nil {
		t.Error(err)
	}
	if !found {
		t.Errorf("Should have returned true, the key should already exist.")
	}
	if oldVal != 3 {
		t.Errorf("Expected old value 3, got: '%v'", oldVal)
	}

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

	firstKey, firstVal, err := node.AtIndex(0)
	if err != nil {
		t.Error(err)
	}
	if firstKey != "" {
		t.Errorf("Expected empty key, got: %v", firstKey)
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

	key, oldVal, err := node.SetIndex(1, "abcdef")
	if err != nil {
		t.Error(err)
	}
	if key != "" {
		t.Errorf("Expected empty key, got: '%v'", key)
	}
	if oldVal != 2.0 {
		t.Errorf("Expected old value 2.0, got: '%v'", oldVal)
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

	oldVal, found, err := node.NK("Z").SetKey("sub2", 3)
	if err == nil {
		t.Errorf("Should have returned an error calling SetKey() on nil")
	}

	oldVal, found, err = node.NKC("Z").SetKey("sub2", 3)
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Errorf("Should have returned false, the key should not already exist.")
	}

	oldVal, found, err = node.NKC("Z").SetKey("sub2", 4)
	if err != nil {
		t.Error(err)
	}
	if !found {
		t.Errorf("Should have returned true, the key should already exist.")
	}
	if oldVal != 3 {
		t.Errorf("Expected old value 3, got: '%v'", oldVal)
	}

	oldVal, found, err = node.NKC("X").NKC("Y").SetKey("sub3", 5)
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Errorf("Should have returned false, the key should not already exist.")
	}

	verifyNodeContent(t, node, `# comment before
b: /* key comment */ {
  sub1: 1  # comment after
} # cm after obj
// Comment B4
a: 2
Z: {
  sub2: 4
}
X: {
  Y: {
    sub3: 5
  }
}
/* Last comment */`)
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

func TestDeclareNodeMap(t *testing.T) {
	var node Node

	node2 := node.NK("a")
	if node2 != nil {
		t.Errorf("node.NK() created a node")
	}

	oldVal, found, err := node.NKC("a").NKC("aa").NKC("aaa").SetKey("aaaa", "a string")
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Errorf("Should have returned false, the key should not already exist.")
	}
	oldVal, found, err = node.SetKey("b", 2)
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Errorf("Should have returned false, the key should not already exist.")
	}
	key, oldVal, err := node.SetIndex(1, 3.0)
	if err != nil {
		t.Error(err)
	}
	if key != "b" {
		t.Errorf("Expected key 'b', got: '%v'", key)
	}
	if oldVal != 2 {
		t.Errorf("Expected old value 2, got: '%v'", oldVal)
	}

	verifyNodeContent(t, &node, `a: {
  aa: {
    aaa: {
      aaaa: a string
    }
  }
}
b: 3`)

	err = node.Append(4)
	if err == nil {
		t.Errorf("Should have returned error when trying to append to a map")
	}
}

func TestDeclareNodeSlice(t *testing.T) {
	var node Node

	node2 := node.NI(0)
	if node2 != nil {
		t.Errorf("node.NI() created a node")
	}

	err := node.Append(13)
	if err != nil {
		t.Error(err)
	}
	err = node.Append("b")
	if err != nil {
		t.Error(err)
	}
	key, oldVal, err := node.SetIndex(1, false)
	if err != nil {
		t.Error(err)
	}
	if key != "" {
		t.Errorf("Expected empty key, got: '%v'", key)
	}
	if oldVal != "b" {
		t.Errorf("Expected old value 'b', got: '%v'", oldVal)
	}

	verifyNodeContent(t, &node, `[
  13
  false
]`)

	node2 = node.NKC("sub")
	if node2 != nil {
		t.Errorf("Should not have been able to create a node by key in a slice")
	}

	_, _, err = node.SetKey("a", 4)
	if err == nil {
		t.Errorf("Should have returned error when trying to set by key on a slice")
	}
}

func TestNodeNoPointer(t *testing.T) {
	txt := `setting1: null  # nada
setting2: true  // yes`

	var node Node
	err := Unmarshal([]byte(txt), &node)
	if err != nil {
		t.Error(err)
	}
	oldVal, found, err := node.SetKey("setting1", 3)
	if err != nil {
		t.Error(err)
	}
	if !found {
		t.Errorf("Should have returned true, the key should already exist")
	}
	if oldVal != nil {
		t.Errorf("Expected old value nil, got: '%v'", oldVal)
	}
	output, err := Marshal(node)
	if err != nil {
		t.Error(err)
	}

	compareStrings(t, output, `{
  setting1: 3  # nada
  setting2: true  // yes
}`)
}

func TestNodeOrderedMapInsertDelete(t *testing.T) {
	txt := `a: 1
b: 2`

	var node Node
	err := Unmarshal([]byte(txt), &node)
	if err != nil {
		t.Error(err)
	}

	key, val, err := node.DeleteIndex(0)
	if err != nil {
		t.Error(err)
	}
	if key != "a" {
		t.Errorf("Expected key 'a', got: '%v'", key)
	}
	if val != 1.0 {
		t.Errorf("Expected old value 1.0, got: '%#v'", val)
	}

	val, found, err := node.Insert(1, key, val)
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Errorf("Found key '%v' that should not have been found", key)
	}

	val, found, err = node.DeleteKey("a")
	if err != nil {
		t.Error(err)
	}
	if !found {
		t.Errorf("Expected to find key 'a' but did not")
	}
	if val != 1.0 {
		t.Errorf("Expected deleted value 1.0, got: '%#v'", val)
	}

	val, found, err = node.Insert(0, "c", 3)
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Error("Found key c that should not have been found")
	}

	verifyNodeContent(t, &node, `c: 3
b: 2`)
}

func TestNodeSliceInsertDelete(t *testing.T) {
	txt := `[
  1
  2
]`

	var node Node
	err := Unmarshal([]byte(txt), &node)
	if err != nil {
		t.Error(err)
	}

	key, val, err := node.DeleteIndex(0)
	if err != nil {
		t.Error(err)
	}
	if key != "" {
		t.Errorf("Expected empty key, got: '%v'", key)
	}
	if val != 1.0 {
		t.Errorf("Expected old value 1.0, got: '%#v'", val)
	}

	val, found, err := node.Insert(1, key, val)
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Errorf("Found key '%v' that should not have been found", key)
	}

	val, found, err = node.Insert(0, "c", 3)
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Error("Found key c that should not have been found")
	}

	// The value '2' has a line break as Cm.After
	verifyNodeContent(t, &node, `[
  3
  2
  1
]`)
}

func TestNodeInsertExistingKey(t *testing.T) {
	txt := `a: 1 // after a
b: 2`

	unmarshal := func(t *testing.T) *Node {
		t.Helper()
		var node Node
		if err := Unmarshal([]byte(txt), &node); err != nil {
			t.Fatal(err)
		}
		return &node
	}

	// Inserting a key that is already there keeps its place, gives back the
	// value it held and reports true, as documented and as SetKey() already
	// did.
	t.Run("into a child node", func(t *testing.T) {
		node := unmarshal(t)
		child := node.Value.(*OrderedMap).Map["a"].(*Node)

		val, found, err := node.Insert(1, "a", 3)
		if err != nil {
			t.Fatal(err)
		}
		if val != 1.0 || !found {
			t.Errorf("Insert returned %v, %v; want 1, true", val, found)
		}
		// The child node found is the one that is given the new value, so that
		// anything pointing at it still points at the value in place.
		if again := node.Value.(*OrderedMap).Map["a"].(*Node); again != child {
			t.Error("Insert replaced the child node instead of using it")
		}
		if child.Value != 3 {
			t.Errorf("Child node holds %#v; want 3", child.Value)
		}
		verifyNodeContent(t, node, `a: 3 // after a
b: 2`)
	})

	// A value that was put in the OrderedMap without a node of its own is
	// wrapped in one and reported as it stands.
	t.Run("of a value that is no child node", func(t *testing.T) {
		node := unmarshal(t)
		om := node.Value.(*OrderedMap)
		om.Map["a"] = 7

		val, found, err := node.Insert(0, "a", 3)
		if err != nil {
			t.Fatal(err)
		}
		if val != 7 || !found {
			t.Errorf("Insert returned %v, %v; want 7, true", val, found)
		}
		if _, isNode := om.Map["a"].(*Node); !isNode {
			t.Errorf("Value is %T; want *hjson.Node", om.Map["a"])
		}
	})

	t.Run("that is not there yet", func(t *testing.T) {
		node := unmarshal(t)

		val, found, err := node.Insert(1, "c", 3)
		if err != nil {
			t.Fatal(err)
		}
		if val != nil || found {
			t.Errorf("Insert returned %v, %v; want nil, false", val, found)
		}
		verifyNodeContent(t, node, `a: 1 // after a
c: 3
b: 2`)
	})

	// An array has no keys to look for, so nothing is ever found in it.
	t.Run("into an array", func(t *testing.T) {
		var node Node
		if err := Unmarshal([]byte("[\n  1\n  2\n]"), &node); err != nil {
			t.Fatal(err)
		}
		val, found, err := node.Insert(0, "a", 3)
		if err != nil {
			t.Fatal(err)
		}
		if val != nil || found {
			t.Errorf("Insert returned %v, %v; want nil, false", val, found)
		}
	})

	// The two functions differ in where they put a key that was missing, not
	// in what they report for one that was there.
	t.Run("reports like SetKey", func(t *testing.T) {
		for _, key := range []string{"a", "b", "not there"} {
			insertVal, insertFound, insertErr := unmarshal(t).Insert(0, key, 3)
			setKeyVal, setKeyFound, setKeyErr := unmarshal(t).SetKey(key, 3)
			if insertErr != nil || setKeyErr != nil {
				t.Fatalf("%s: %v, %v", key, insertErr, setKeyErr)
			}
			if insertVal != setKeyVal || insertFound != setKeyFound {
				t.Errorf("%s: Insert returned %v, %v; SetKey returned %v, %v",
					key, insertVal, insertFound, setKeyVal, setKeyFound)
			}
		}
	})
}
