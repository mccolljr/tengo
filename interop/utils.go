package interop

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/d5/tengo"
)

var (
	rtString = reflect.TypeOf("")
	rtBytes  = reflect.TypeOf([]byte(nil))
)

func convert(from tengo.Object, to reflect.Type) reflect.Value {
	if goObj, isGoObj := from.(*GoObject); isGoObj {
		if goObj.V.Type().AssignableTo(to) || to.AssignableTo(goObj.V.Type()) {
			return goObj.V.Convert(to)
		}

		return reflect.Value{}
	}

	switch to.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if i, ok := tengo.ToInt64(from); ok {
			return reflect.ValueOf(i).Convert(to)
		}

	case reflect.Float32, reflect.Float64:
		if f, ok := tengo.ToFloat64(from); ok {
			return reflect.ValueOf(f).Convert(to)
		}

	case reflect.Bool:
		return reflect.ValueOf(!from.IsFalsy()).Convert(to)

	case reflect.String:
		if s, ok := tengo.ToString(from); ok {
			return reflect.ValueOf(s).Convert(to)
		}

	case reflect.Slice:
		if to.AssignableTo(rtBytes) {
			if b, ok := tengo.ToByteSlice(from); ok {
				return reflect.ValueOf(b).Convert(to)
			}
		}

		var elems []tengo.Object
		switch real := from.(type) {
		case *tengo.Array:
			elems = real.Value
		case *tengo.ImmutableArray:
			elems = real.Value
		default:
			return reflect.Value{}
		}

		newVal := reflect.MakeSlice(to, len(elems), len(elems))
		elemType := to.Elem()
		for i := 0; i < len(elems); i++ {
			v := convert(elems[i], elemType)
			if !v.IsValid() {
				return reflect.Value{}
			}
			newVal.Index(i).Set(v)
		}
		return newVal

	case reflect.Map:
		if !rtString.AssignableTo(to.Key()) {
			return reflect.Value{}
		}

		var elems map[string]tengo.Object
		switch real := from.(type) {
		case *tengo.Map:
			elems = real.Value
		case *tengo.ImmutableMap:
			elems = real.Value
		default:
			return reflect.Value{}
		}

		newVal := reflect.MakeMap(to)
		elemTyp := to.Elem()
		keyTyp := to.Key()
		for k, v := range elems {
			elemVal := convert(v, elemTyp)
			if !elemVal.IsValid() {
				return reflect.Value{}
			}
			newVal.SetMapIndex(reflect.ValueOf(k).Convert(keyTyp), elemVal)
		}
		return newVal
	}

	return reflect.Value{}
}

func setValue(v reflect.Value, src tengo.Object) error {
	if !v.CanSet() {
		if v.Kind() != reflect.Ptr {
			return errors.New("value cannot be set")
		}
		return setValue(v.Elem(), src)
	}

	if converted := convert(src, v.Type()); converted.IsValid() {
		v.Set(converted)
		return nil
	}

	return fmt.Errorf("value of type %s cannot be set with %s", v.Type(), src.TypeName())
}

func indexSet(v reflect.Value, key, val tengo.Object) error {
	if v.Kind() == reflect.Ptr {
		return indexSet(v.Elem(), key, val)
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		index, ok := tengo.ToInt(key)
		if !ok {
			return tengo.ErrInvalidIndexType
		}

		if v.Kind() == reflect.Slice && v.IsNil() {
			return errors.New("cannot set value in nil slice")
		}

		if index < 0 || index >= v.Len() {
			return errors.New("index out of bounds")
		}

		target := v.Index(index)
		if !target.CanSet() {
			return fmt.Errorf("cannot index-set on %s", v.Type())
		}

		return setValue(target, val)

	case reflect.Map:
		kval := convert(key, v.Type().Key())
		if !kval.IsValid() {
			return tengo.ErrInvalidIndexType
		}

		vval := convert(val, v.Type().Elem())
		if !vval.IsValid() {
			return &tengo.ErrInvalidArgumentType{
				Name:     "value",
				Expected: v.Type().Key().String(),
				Found:    val.TypeName(),
			}
		}

		if v.IsNil() {
			return errors.New("cannot set value in nil map")
		}

		v.SetMapIndex(kval, vval)
		return nil

	case reflect.Struct:
		str, ok := tengo.ToString(key)
		if !ok {
			return tengo.ErrInvalidIndexType
		}

		if f := v.FieldByName(str); f.IsValid() {
			if newVal := convert(val, f.Type()); newVal.IsValid() {
				f.Set(newVal)
				return nil
			}
		}
	}

	return fmt.Errorf("cannot index-set type %s", v.Type())
}

func indexGet(v reflect.Value, key tengo.Object) (tengo.Object, error) {
	if str, ok := tengo.ToString(key); ok {
		if m := v.MethodByName(str); m.IsValid() {
			return FromValue(m), nil
		}
	}

	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Array:
		index, ok := tengo.ToInt(key)
		if !ok {
			return nil, tengo.ErrInvalidIndexType
		}

		if v.IsNil() || index < 0 || index >= v.Len() {
			return tengo.UndefinedValue, nil
		}

		return FromValue(v.Index(index)), nil

	case reflect.Map:
		k := convert(key, v.Type().Key())
		if !k.IsValid() {
			return nil, tengo.ErrInvalidIndexType
		}

		if v.IsNil() {
			return tengo.UndefinedValue, nil
		}

		return FromValue(v.MapIndex(k)), nil

	case reflect.Struct:
		str, ok := tengo.ToString(key)
		if !ok {
			return nil, tengo.ErrInvalidIndexType
		}

		if f := v.FieldByName(str); f.IsValid() {
			return FromValue(f), nil
		}

	case reflect.Ptr:
		return indexGet(v.Elem(), key)
	}

	return nil, fmt.Errorf("cannot index type %s", v.Type())
}
