package runtime_test

import (
	"testing"

	"github.com/d5/tengo/objects"
)

func callTengoFunc(args ...objects.Object) (objects.Object, error) {
	if len(args) < 1 {
		return nil, objects.ErrWrongNumArguments
	}

	fn, isCallable := args[0].(objects.Callable)
	if !isCallable {
		return nil, objects.ErrInvalidArgumentType{
			"fn", "compiled-function or closure", args[0].TypeName(),
		}
	}

	return fn.Call(args[1:]...)
}

var interopOpts = &testopts{
	symbols: map[string]objects.Object{
		"go_call": &objects.UserFunction{
			Name:  "go_call",
			Value: callTengoFunc,
		},
	},
	//modules:   stdlib.GetModuleMap(stdlib.AllModuleNames()...),
	maxAllocs:   2048,
	skip2ndPass: true,
}

func TestGoInterop(t *testing.T) {
	expect(t, `
		fn := func() { return [1,2,3]; }
		out = go_call(fn)`,
		interopOpts, ARR{1, 2, 3})

	expect(t, `
		fn := func() {
			x := [1,2,3]
			return func() {
				return x
			}
		}()
		out = go_call(fn)`,
		interopOpts, ARR{1, 2, 3})

	expect(t, `
		fn := func(...args) {
			return args
		}
		out = go_call(fn, 1, 2, 3)`,
		interopOpts, ARR{1, 2, 3})

	expect(t, `
		// regular recursion
		sum_to_zero := func(x) {
			if x == 1 {
				return 1
			}
			return x + sum_to_zero(x-1)
		}
		out = go_call(sum_to_zero, 10)`,
		interopOpts, 10+9+8+7+6+5+4+3+2+1)

	expect(t, `
		// tail-call recursion
		sum_to_zero := func(x) {
			total := 0
			tail_call_recurser := func() {
				if x == 0 {
					return total
				}
				total += x
				x--
				return tail_call_recurser()
			}
			tail_call_recurser()
			return total
		}
		out = go_call(sum_to_zero, 10)`,
		interopOpts, 10+9+8+7+6+5+4+3+2+1)

	expect(t, `
		call_from_go := func(fn, arg) {
			return go_call(fn, arg)
		}

		do_stuff := func(x) { return [x] }
		out = go_call(call_from_go, do_stuff, 11)`,
		interopOpts, ARR{11})

	expect(t, `
		call_from_go := func(fn, arg) { return go_call(fn, arg) }
		do_stuff := func(x) { return [x] }
		out = go_call(call_from_go, do_stuff, 11)`,
		interopOpts, ARR{11})

	expect(t, `
		out = 0
		increment := func() { out++ }
		go_call(increment)
		increment()
		go_call(increment)
		increment()`, interopOpts, 4)

	expectError(t, `
		div_by_zero := func(x) { return x/0 }
		go_call(div_by_zero, 1)`,
		interopOpts, "Runtime Error: runtime error: integer divide by zero")

	expectError(t, `
		call_from_go := func(fn, arg) { return go_call(fn, arg) }
		div_by_zero := func(x) { return x/0 }
		go_call(call_from_go, div_by_zero, 1)`,
		interopOpts, "Runtime Error: runtime error: integer divide by zero")
}
