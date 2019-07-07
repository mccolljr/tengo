package interop

import (
	"fmt"
	"reflect"

	"github.com/d5/tengo/compiler/token"

	"github.com/d5/tengo"
)

func FromValue(v reflect.Value) tengo.Object {
	if !v.IsValid() {
		return tengo.UndefinedValue
	}

	obj, err := tengo.FromInterface(v.Interface())
	if err == nil {
		return obj
	}

	return &GoObject{
		V: v,
	}
}

func FromInterface(src interface{}) tengo.Object {
	return FromValue(reflect.ValueOf(src))
}

type GoObject struct {
	V reflect.Value
}

func (o *GoObject) TypeName() string { return "go:" + o.V.Type().String() }

func (o *GoObject) String() string { return fmt.Sprintf("%v", o.V) }

func (o *GoObject) Copy() tengo.Object { return o }

func (o *GoObject) IndexGet(key tengo.Object) (tengo.Object, error) {
	return indexGet(o.V, key)
}

func (o *GoObject) IndexSet(key, val tengo.Object) error {
	return indexSet(o.V, key, val)
}

func (o *GoObject) BinaryOp(op token.Token, rhs tengo.Object) (tengo.Object, error) {
	return nil, tengo.ErrInvalidOperator
}

func (o *GoObject) IsFalsy() bool {
	switch o.V.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface, reflect.Func:
		return o.V.IsNil()
	default:
		return o.V.Interface() == reflect.Zero(o.V.Type()).Interface()
	}
}

func (o *GoObject) Equals(other tengo.Object) bool {
	return other == o
}

func (o *GoObject) CanCall() bool { return o.V.Kind() == reflect.Func }
func (o *GoObject) Call(rt tengo.Interop, args ...tengo.Object) (tengo.Object, error) {
	if !o.CanCall() {
		return nil, fmt.Errorf("%s is not callable", o.t)
	}

	typ := o.V.Type()
	numIn := typ.NumIn()
	if len(args) != numIn {
		return nil, tengo.ErrWrongNumArguments
	}

	ins := make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		argTyp := typ.In(i)
		arg := convert(args[i], argTyp)
		if !arg.IsValid() {
			return nil, tengo.ErrInvalidArgumentType{
				Name:     fmt.Sprintf("argument %d", i),
				Expected: argTyp.String(),
				Found:    args[i].TypeName(),
			}
		}
		ins[i] = arg
	}
	result := o.V.Call(ins)
	if len(result) == 0 {
		return tengo.UndefinedValue, nil
	}

	if len(result) == 1 {
		return FromValue(result[0]), nil
	}

	outs := make([]tengo.Object, len(result))
	for i := 0; i < len(result); i++ {
		outs[i] = FromValue(result[i])
	}
	return &tengo.Array{
		Value: outs,
	}, nil
}

func (o *GoObject) CanIterate() bool {
	switch o.V.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return true
	case reflect.Chan:
		return o.V.Type().ChanDir() != reflect.SendDir
	}
	return false
}

func (o *GoObject) Iterate() tengo.Iterator {
	switch o.V.Kind() {
	case reflect.Map:
		return &MapIterator{
			iter: *o.V.MapRange(),
		}
	case reflect.Array, reflect.Slice, reflect.String:
		return &IndexIterator{
			slice: o.V,
			index: -1,
		}
	case reflect.Chan:
		if o.V.Type().ChanDir() == reflect.SendDir {
			panic("cannot iterate over send-only channel")
		}

		return &ChannelIterator{
			chanv: o.v,
			count: -1,
		}
	default:
		panic(fmt.Sprintf("cannot iterate value of type %s", o.t.Name()))
	}
}

func (o *GoObject) CanSpread() bool        { return false }
func (o *GoObject) Spread() []tengo.Object { return nil }
