package runtime_test

import (
	"fmt"
	"testing"

	"github.com/d5/tengo/stdlib"

	"github.com/d5/tengo/objects"
)

func testNativeFn(args ...objects.Object) (objects.Object, error) {
	fmt.Println(args)
	return &objects.Int{Value: int64(len(args))}, nil
}

func TestInterop(t *testing.T) {
	expect(t, `
fmt := import("fmt")
a := func(){
	b := (func(){
		return go_func(1,2,3,4)
	})()
	c := go_func("test")
	return [b, c]
}
d := a()
x := fmt.println(d)
fmt.println(type_name(x))
out = d`,
		&testopts{
			maxAllocs: 2048,
			symbols: map[string]objects.Object{
				"go_func": &objects.UserFunction{Name: "go_func", Value: testNativeFn},
			},
			skip2ndPass: true,
			modules:     stdlib.GetModuleMap("fmt"),
		}, &objects.Array{
			Value: []objects.Object{&objects.Int{Value: 4}, &objects.Int{Value: 1}},
		})
}
