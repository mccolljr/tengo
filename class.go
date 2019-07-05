package tengo

import (
	"errors"
	"fmt"
)

type Class struct {
	ObjectImpl
	Base *Class
	Name string
	Body map[string]Object
}

func (o *Class) TypeName() string {
	return "class"
}

func (o *Class) String() string {
	return fmt.Sprintf("<class %s>", o.Name)
}

func (o *Class) Copy() Object {
	return o
}

func (o *Class) initInstance(
	rt Interop, self ClassInstanceSelf, constructors *[]Object,
) {
	if o.Base != nil {
		o.Base.initInstance(rt, self, constructors)
	}

	for k, v := range o.Body {
		// private methods & properties
		if k != "" && k[0] == '_' {
			if v.CanCall() {
				bound, _ := BuiltinBind(rt, v.Copy(), self)
				self.Private[k] = bound
				continue
			}

			self.Private[k] = v.Copy()
			continue
		}

		// public methods & properties
		if v.CanCall() {
			if k == "init" {
				*constructors = append(*constructors, v)
			} else {
				bound, _ := BuiltinBind(rt, v.Copy(), self)
				self.Public[k] = bound
			}
			continue
		}

		self.Public[k] = v.Copy()
	}
}

func (o *Class) IndexGet(key Object) (Object, error) {
	switch str, _ := ToString(key); str {
	case "name":
		return &String{Value: o.Name}, nil
	default:
		return UndefinedValue, nil
	}
}

func (o *Class) Call(rt Interop, args ...Object) (Object, error) {
	inst := &ClassInstance{
		Class:   o,
		Public:  map[string]Object{},
		Private: map[string]Object{},
	}

	self := ClassInstanceSelf{ClassInstance: inst}
	constructors := []Object(nil)
	o.initInstance(rt, self, &constructors)

	initArgs := append([]Object{self}, args...)
	for _, c := range constructors {
		ret, err := rt.InteropCall(c, initArgs...)
		if err != nil {
			return nil, err
		}

		if e, _ := ret.(*Error); e != nil {
			es, _ := Format("%v", e)
			return nil, errors.New(es)
		}
	}

	return inst, nil
}

func (o *Class) CanCall() bool {
	return true
}
