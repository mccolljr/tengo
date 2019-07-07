package interop

import (
	"reflect"
	"strings"
	"testing"

	"github.com/d5/tengo"
)

func TestSetValue(t *testing.T) {
	cases := []struct {
		target      interface{}
		src         tengo.Object
		expected    interface{}
		expectedErr string
	}{
		{new(string), tengoize("test"), "test", ""},

		{new([]byte), tengoize([]byte("abc")), []byte("abc"), ""},
		{new([]byte), tengoize(ARR{1, 2, 3}), []byte{1, 2, 3}, ""},
		{new([]byte), tengoize("fromString"), []byte("fromString"), ""},

		{new(int), tengoize(-1), int(-1), ""},
		{new(int8), tengoize(-8), int8(-8), ""},
		{new(int16), tengoize(-16), int16(-16), ""},
		{new(int32), tengoize(-32), int32(-32), ""},
		{new(int64), tengoize(-64), int64(-64), ""},

		{new(uint), tengoize(1), uint(1), ""},
		{new(uint8), tengoize(8), uint8(8), ""},
		{new(uint16), tengoize(16), uint16(16), ""},
		{new(uint32), tengoize(32), uint32(32), ""},
		{new(uint64), tengoize(64), uint64(64), ""},

		{new(float32), tengoize(32.0), float32(32.0), ""},
		{new(float64), tengoize(64.0), float64(64.0), ""},

		{new(bool), tengoize(false), false, ""},
		{new(bool), tengoize(true), true, ""},
		{new(bool), tengoize(nil), false, ""},
		{new(bool), tengoize(0), false, ""},
		{new(bool), tengoize(""), false, ""},

		{new([]int), tengoize(ARR{1, 2, 3}), []int{1, 2, 3}, ""},
		{new(map[string]int), tengoize(MAP{"a": 1, "b": 2}), map[string]int{"a": 1, "b": 2}, ""},

		{new(struct{ int }), FromInterface(struct{ int }{int: 10}), struct{ int }{10}, ""},
	}

	for _, c := range cases {
		tval := reflect.ValueOf(c.target)
		err := setValue(tval, c.src)

		if c.expectedErr != "" {
			if err == nil || strings.Index(err.Error(), c.expectedErr) == -1 {
				t.Errorf("expected error like %q, got nil", c.expectedErr)
			}
			continue
		}

		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}

		if !reflect.DeepEqual(tval.Elem().Interface(), c.expected) {
			t.Errorf("expected %v, got %v", c.expected, tval)
		}
	}
}
