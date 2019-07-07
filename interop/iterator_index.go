package interop

import (
	"reflect"

	"github.com/d5/tengo"
)

type IndexIterator struct {
	tengo.ObjectImpl
	slice reflect.Value
	index int
	value reflect.Value
}

func (o *IndexIterator) Next() bool {
	if o.index >= o.slice.Len()-1 {
		return false
	}
	o.index++
	o.value = o.slice.Index(o.index)
	return true
}

func (o *IndexIterator) Key() tengo.Object {
	return FromInterface(o.index)
}

func (o *IndexIterator) Value() tengo.Object {
	return FromValue(o.value)
}
