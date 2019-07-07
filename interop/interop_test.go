package interop

import (
	"reflect"
	"strings"
	"testing"

	"github.com/d5/tengo"
	"github.com/d5/tengo/script"
	"github.com/d5/tengo/stdlib"
)

type (
	ARR = []interface{}
	MAP = map[string]interface{}
)

func expect(t *testing.T, code string, inputs MAP, want interface{}) {
	s := script.New([]byte(code))

	for name, val := range inputs {
		if err := s.Add(name, FromInterface(val)); err != nil {
			t.Error(err)
			return
		}
	}

	if err := s.Add("out", nil); err != nil {
		t.Error(err)
		return
	}

	s.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

	cmp, err := s.Run()
	if err != nil {
		t.Error(err)
		return
	}

	if goObj, ok := cmp.Get("out").Object().(*GoObject); ok {
		if !reflect.DeepEqual(goObj.v.Interface(), want) {
			t.Logf("%+v != %+v\n", goObj.v.Interface(), want)
			t.Errorf("expected %[1]v (%[1]T), got %[2]v (%[2]T)", want, goObj.v.Interface())
		}
		return
	}

	wantObj := FromInterface(want)
	if !reflect.DeepEqual(cmp.Get("out").Object(), wantObj) {
		t.Logf("%+v != %+v\n", cmp.Get("out").Object(), wantObj)
		t.Errorf("expected %[1]v (%[1]T), got %[2]v (%[2]T)", wantObj, cmp.Get("out").Object())
	}
}

func expectError(t *testing.T, code string, inputs MAP, wantErr string) {
	s := script.New([]byte(code))

	for name, val := range inputs {
		if err := s.Add(name, FromInterface(val)); err != nil {
			t.Error(err)
			return
		}
	}

	s.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

	_, err := s.Run()
	if err == nil {
		t.Errorf("expected error like %q, got nil", wantErr)
		return
	}

	if strings.Index(err.Error(), wantErr) < 0 {
		t.Errorf("expected error like %q, got %s", wantErr, err)
		return
	}
}

func tengoize(src interface{}) tengo.Object {
	obj, err := tengo.FromInterface(src)
	if err != nil {
		panic(err)
	}
	return obj
}
