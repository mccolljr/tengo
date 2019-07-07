package interop

import (
	"errors"
	"testing"
)

type TestObjectWithBehavior struct {
	Field1 string
	Field2 string
}

func (t TestObjectWithBehavior) DoStuff() error {
	return errors.New("test")
}

func (t TestObjectWithBehavior) DoStuff2() string {
	return "test!"
}

func indexTestInputs() MAP {
	return MAP{
		"test_obj":   &TestObjectWithBehavior{},
		"test_map":   map[string]int{},
		"test_slice": []int{1, 2, 3},
	}
}

func TestIndexGet(t *testing.T) {

	expect(t, `
		out = [test_obj.Field1, test_obj.Field1];
	`, indexTestInputs(), ARR{"", ""})

	expect(t, `
		out = test_obj.DoStuff()
	`, indexTestInputs(), errors.New("test"))

	expect(t, `
		out = test_obj.DoStuff2()
	`, indexTestInputs(), "test!")
}

func TestIndexSet(t *testing.T) {
	expect(t, `
		test_obj.Field1 = "test";
		out = test_obj.Field1;
	`, indexTestInputs(), "test")

	expect(t, `
		test_map.a_value = 1;
		out = test_map.a_value;
	`, indexTestInputs(), 1)

	expect(t, `
		test_slice[0] = -5;
		out = test_slice[0];
	`, indexTestInputs(), -5)

	expectError(t, `
	test_slice[-1] = 0
	`, indexTestInputs(), "index out of bounds")

	expectError(t, `
	test_slice[3] = 0
	`, indexTestInputs(), "index out of bounds")
}
