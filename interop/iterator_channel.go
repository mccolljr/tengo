package interop

import (
	"reflect"

	"github.com/d5/tengo"
)

type ChannelIterator struct {
	tengo.ObjectImpl
	chanv reflect.Value
	count int
	value reflect.Value
}

func (o *ChannelIterator) Next() bool {
	got, ok := o.chanv.Recv()
	o.value = got
	o.count++
	return ok
}

func (o *ChannelIterator) Key() tengo.Object {
	return FromInterface(o.count)
}

func (o *ChannelIterator) Value() tengo.Object {
	return FromValue(o.value)
}
