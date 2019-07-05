package tengo

import (
	"fmt"

	"github.com/d5/tengo/compiler/token"
)

// Builtins contains all default builtin functions.
// Use GetBuiltinFunctions instead of accessing Builtins directly.
var Builtins = []*BuiltinFunction{
	{Name: "len", Value: BuiltinLen},
	{Name: "copy", Value: BuiltinCopy},
	{Name: "append", Value: BuiltinAppend},
	{Name: "string", Value: BuiltinString},
	{Name: "int", Value: BuiltinInt},
	{Name: "bool", Value: BuiltinBool},
	{Name: "float", Value: BuiltinFloat},
	{Name: "char", Value: BuiltinChar},
	{Name: "bytes", Value: BuiltinBytes},
	{Name: "time", Value: BuiltinTime},
	{Name: "is_int", Value: BuiltinIsInt},
	{Name: "is_float", Value: BuiltinIsFloat},
	{Name: "is_string", Value: BuiltinIsString},
	{Name: "is_bool", Value: BuiltinIsBool},
	{Name: "is_char", Value: BuiltinIsChar},
	{Name: "is_bytes", Value: BuiltinIsBytes},
	{Name: "is_array", Value: BuiltinIsArray},
	{Name: "is_immutable_array", Value: BuiltinIsImmutableArray},
	{Name: "is_map", Value: BuiltinIsMap},
	{Name: "is_immutable_map", Value: BuiltinIsImmutableMap},
	{Name: "is_iterable", Value: BuiltinIsIterable},
	{Name: "is_time", Value: BuiltinIsTime},
	{Name: "is_error", Value: BuiltinIsError},
	{Name: "is_undefined", Value: BuiltinIsUndefined},
	{Name: "is_function", Value: BuiltinIsFunction},
	{Name: "is_callable", Value: BuiltinIsCallable},
	{Name: "type_name", Value: BuiltinTypeName},
	{Name: "format", Value: BuiltinFormat},
	{Name: "bind", Value: BuiltinBind},
	{Name: token.Class.String(), Value: BuiltinClass},
}

// len(obj object) => int
func BuiltinLen(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	switch arg := args[0].(type) {
	case *Array:
		return &Int{Value: int64(len(arg.Value))}, nil
	case *ImmutableArray:
		return &Int{Value: int64(len(arg.Value))}, nil
	case *String:
		return &Int{Value: int64(len(arg.Value))}, nil
	case *Bytes:
		return &Int{Value: int64(len(arg.Value))}, nil
	case *Map:
		return &Int{Value: int64(len(arg.Value))}, nil
	case *ImmutableMap:
		return &Int{Value: int64(len(arg.Value))}, nil
	default:
		return nil, ErrInvalidArgumentType{
			Name:     "first",
			Expected: "array/string/bytes/map",
			Found:    arg.TypeName(),
		}
	}
}

// append(arr, items...)
func BuiltinAppend(_ Interop, args ...Object) (Object, error) {
	if len(args) < 2 {
		return nil, ErrWrongNumArguments
	}

	switch arg := args[0].(type) {
	case *Array:
		return &Array{Value: append(arg.Value, args[1:]...)}, nil
	case *ImmutableArray:
		return &Array{Value: append(arg.Value, args[1:]...)}, nil
	default:
		return nil, ErrInvalidArgumentType{
			Name:     "first",
			Expected: "array",
			Found:    arg.TypeName(),
		}
	}
}

func BuiltinString(_ Interop, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*String); ok {
		return args[0], nil
	}

	v, ok := ToString(args[0])
	if ok {
		if len(v) > MaxStringLen {
			return nil, ErrStringLimit
		}

		return &String{Value: v}, nil
	}

	if argsLen == 2 {
		return args[1], nil
	}

	return UndefinedValue, nil
}

func BuiltinInt(_ Interop, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Int); ok {
		return args[0], nil
	}

	v, ok := ToInt64(args[0])
	if ok {
		return &Int{Value: v}, nil
	}

	if argsLen == 2 {
		return args[1], nil
	}

	return UndefinedValue, nil
}

func BuiltinFloat(_ Interop, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Float); ok {
		return args[0], nil
	}

	v, ok := ToFloat64(args[0])
	if ok {
		return &Float{Value: v}, nil
	}

	if argsLen == 2 {
		return args[1], nil
	}

	return UndefinedValue, nil
}

func BuiltinBool(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Bool); ok {
		return args[0], nil
	}

	v, ok := ToBool(args[0])
	if ok {
		if v {
			return TrueValue, nil
		}

		return FalseValue, nil
	}

	return UndefinedValue, nil
}

func BuiltinChar(_ Interop, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Char); ok {
		return args[0], nil
	}

	v, ok := ToRune(args[0])
	if ok {
		return &Char{Value: v}, nil
	}

	if argsLen == 2 {
		return args[1], nil
	}

	return UndefinedValue, nil
}

func BuiltinBytes(_ Interop, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}

	// bytes(N) => create a new bytes with given size N
	if n, ok := args[0].(*Int); ok {
		if n.Value > int64(MaxBytesLen) {
			return nil, ErrBytesLimit
		}

		return &Bytes{Value: make([]byte, int(n.Value))}, nil
	}

	v, ok := ToByteSlice(args[0])
	if ok {
		if len(v) > MaxBytesLen {
			return nil, ErrBytesLimit
		}

		return &Bytes{Value: v}, nil
	}

	if argsLen == 2 {
		return args[1], nil
	}

	return UndefinedValue, nil
}

func BuiltinTime(_ Interop, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Time); ok {
		return args[0], nil
	}

	v, ok := ToTime(args[0])
	if ok {
		return &Time{Value: v}, nil
	}

	if argsLen == 2 {
		return args[1], nil
	}

	return UndefinedValue, nil
}

func BuiltinCopy(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	return args[0].Copy(), nil
}

func BuiltinFormat(_ Interop, args ...Object) (Object, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return nil, ErrWrongNumArguments
	}

	format, ok := args[0].(*String)
	if !ok {
		return nil, ErrInvalidArgumentType{
			Name:     "format",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
	}
	if numArgs == 1 {
		// okay to return 'format' directly as String is immutable
		return format, nil
	}

	s, err := Format(format.Value, args[1:]...)
	if err != nil {
		return nil, err
	}

	return &String{Value: s}, nil
}

func BuiltinTypeName(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	return &String{Value: args[0].TypeName()}, nil
}

func BuiltinIsString(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*String); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsInt(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Int); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsFloat(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Float); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsBool(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Bool); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsChar(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Char); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsBytes(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Bytes); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsArray(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Array); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsImmutableArray(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*ImmutableArray); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsMap(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Map); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsImmutableMap(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*ImmutableMap); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsTime(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Time); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsError(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if _, ok := args[0].(*Error); ok {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsUndefined(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if args[0] == UndefinedValue {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsFunction(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	switch args[0].(type) {
	case *CompiledFunction:
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsCallable(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if args[0].CanCall() {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinIsIterable(_ Interop, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	if args[0].CanIterate() {
		return TrueValue, nil
	}

	return FalseValue, nil
}

func BuiltinBind(_ Interop, args ...Object) (Object, error) {
	if len(args) == 0 {
		return nil, ErrWrongNumArguments
	}

	return &GoFunction{
		Value: func(rt Interop, newArgs ...Object) (ret Object, err error) {
			return rt.InteropCall(args[0], append(args[1:], newArgs...)...)
		},
	}, nil
}

func BuiltinClass(rt Interop, args ...Object) (Object, error) {
	numArgs := len(args)
	if numArgs < 2 || numArgs > 3 {
		return nil, ErrWrongNumArguments
	}

	base := (*Class)(nil)
	name := args[0]
	body := args[1]
	if numArgs == 3 {
		c, ok := args[0].(*Class)
		if !ok {
			return nil, fmt.Errorf("class: extended object must be a class")
		}
		base = c
		name = args[1]
		body = args[2]
	}

	nameStr, _ := ToString(name)
	if nameStr == "" {
		return nil, fmt.Errorf("class: name must be non-empty string")
	}

	bodyMap := map[string]Object{}

	switch real := body.(type) {
	case *Map:
		for k, v := range real.Value {
			bodyMap[k] = v.Copy()
		}
	case *ImmutableMap:
		for k, v := range real.Value {
			bodyMap[k] = v.Copy()
		}
	default:
		return nil, fmt.Errorf("class: body must be map or immutable-map")
	}

	return &Class{
		Base: base,
		Name: nameStr,
		Body: bodyMap,
	}, nil
}
