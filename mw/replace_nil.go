package mw

import (
	"reflect"
)

// ReplaceNil initialization nil values. An addressable type must be passed in; even if it's a pointer type,
// its address must be passed. e.g., a := &A{} requires &a to be passed, not a .
func ReplaceNil(data any) {
	v := reflect.ValueOf(data)

	// Replacement can only occur if IsValid is true and the value is addressable
	if !v.IsValid() || v.Kind() != reflect.Ptr {
		return
	}
	replaceNil(v)
}

func replaceNil(v reflect.Value) {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
			// If the struct itself is nil, initialize it directly without recursion
			if v.Elem().Kind() == reflect.Struct {
				return
			}
		}
		replaceNil(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			replaceNil(f)
		}
	case reflect.Slice:
		if v.IsNil() {
			v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		}
	case reflect.Map:
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
	default:
		return
	}
}
