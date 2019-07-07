package interop

import (
	"reflect"

	"github.com/d5/tengo"
)

type MapIterator struct {
	tengo.ObjectImpl
	iter reflect.MapIter
}

func (o *MapIterator) Next() bool { return o.iter.Next() }

func (o *MapIterator) Key() tengo.Object {
	return FromValue(o.iter.Key())
}

func (o *MapIterator) Value() tengo.Object {
	return FromValue(o.iter.Value())
}
