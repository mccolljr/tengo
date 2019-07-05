package tengo

import (
	"fmt"
)

type ClassInstance struct {
	ObjectImpl
	Class   *Class
	Public  map[string]Object
	Private map[string]Object
}

func (o *ClassInstance) TypeName() string {
	return o.Class.Name
}

func (o *ClassInstance) String() string {
	return fmt.Sprintf("<%s %p>", o.TypeName(), o)
}

func (o *ClassInstance) IndexGet(key Object) (Object, error) {
	str, _ := ToString(key)
	if str == "" {
		return UndefinedValue, nil
	}

	if str[0] == '_' {
		// ignore attempts to read private propreties
		return UndefinedValue, nil
	}

	val := o.Public[str]
	if val == nil {
		val = UndefinedValue
	}

	return val, nil
}

func (o *ClassInstance) IndexSet(key, val Object) error {
	str, _ := ToString(key)
	if str == "" {
		return nil
	}

	if str[0] == '_' {
		// ignore attempts to set private propreties
		return nil
	}

	o.Public[str] = val
	return nil
}

func (o *ClassInstance) Copy() Object {
	// TODO actually make copy
	return o
}

type ClassInstanceSelf struct {
	*ClassInstance
}

func (o ClassInstanceSelf) IndexGet(key Object) (Object, error) {
	str, _ := ToString(key)
	if str == "" {
		return UndefinedValue, nil
	}

	if str[0] == '_' {
		val := o.Private[str]
		if val == nil {
			val = UndefinedValue
		}
		return val, nil
	}

	val := o.Public[str]
	if val == nil {
		val = UndefinedValue
	}

	return val, nil
}

func (o ClassInstanceSelf) IndexSet(key, val Object) error {
	str, _ := ToString(key)
	if str == "" {
		return nil
	}

	if str[0] == '_' {
		o.Private[str] = val
		return nil
	}

	o.Public[str] = val
	return nil
}
