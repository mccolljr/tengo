package interop

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

func TestCall(t *testing.T) {
	funcs := MAP{
		"inc": func(a int) int {
			return a + 1
		},
		"dec": func(a int) int {
			return a - 1
		},
		"concat": func(a, b string) string {
			return a + b
		},
		"split": func(a, sep string) []string {
			return strings.Split(a, sep)
		},
		"multi": func(a int) (b, c, d int) {
			return a + 1, a + 2, a + 3
		},
		"new_reader": func(s string) io.Reader {
			return strings.NewReader(s)
		},
		"read_all": func(r io.Reader) interface{} {
			data, err := ioutil.ReadAll(r)
			if err != nil {
				return err
			}
			return data
		},
	}

	expect(t, `out = inc(1)`, funcs, 2)
	expect(t, `out = dec(1)`, funcs, 0)
	expect(t, `out = dec(12.5)`, funcs, 11)
	expect(t, `out = concat("a", "b")`, funcs, "ab")
	expect(t, `out = split("a b c", " ")`, funcs, []string{"a", "b", "c"})
	expect(t, `out = read_all(new_reader("test"))`, funcs, []byte("test"))
	expect(t, `out = type_name(new_reader(""))`, funcs, "go:io.Reader")
	expect(t, `out = multi(1)`, funcs, ARR{2, 3, 4})

	expectError(t, `inc()`, funcs, "wrong number of arguments")
	expectError(t, `inc(1,2)`, funcs, "wrong number of arguments")
	expectError(t, `inc("test")`, funcs, "expected int, found string")
}
