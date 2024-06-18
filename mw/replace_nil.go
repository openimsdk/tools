package mw

import (
	"github.com/openimsdk/tools/utils/datautil"
	"reflect"
)

// ReplaceNil initialization nil values. An addressable type must be passed in; even if it's a pointer type,
// its address must be passed. e.g., a := &A{} requires &a to be passed, not a .
func ReplaceNil(data *any) {
	v := reflect.ValueOf(data)

	// Replacement can only occur if IsValid is true and the value is addressable
	if !v.IsValid() || v.Kind() != reflect.Ptr {
		return
	}

	replaceNil(v)
}

func replaceNil(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			// Handle multi-level pointers
			if v.Type().Elem().Kind() == reflect.Pointer {
				v.Set(reflect.New(v.Type().Elem()))
			}
		}
		replaceNil(v.Elem())
	case reflect.Slice:
		if v.IsNil() {
			v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		}
	case reflect.Map:
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			replaceNil(field)
		}
	case reflect.Interface:
		if !v.IsNil() && !shouldReplace(v) {
			// If the interface is already initialized, recursively replace the internal nils
			replaceNil(v.Elem())
		} else {
			// If the interface is not initialized, the struct will be initialized as {}
			switch getRealType(v.Interface()) {
			case reflect.Slice:
				v.Set(reflect.MakeSlice(v.Type(), 0, 0))
			case reflect.Map:
				v.Set(reflect.MakeMap(v.Type()))
			case reflect.Struct:
				v.Set(reflect.New(reflect.TypeOf(struct{}{})))
			default:
			}
		}
	default:
		return
	}
}

func getRealType(data any) reflect.Kind {
	t := reflect.TypeOf(data)
	if t == nil {
		return reflect.Invalid
	}
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	return t.Kind()
}

func shouldReplace(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	if datautil.Contain(v.Kind(), []reflect.Kind{reflect.Slice, reflect.Map}...) && v.IsNil() {
		return true
	}
	switch v.Kind() {
	case reflect.Ptr:
		return shouldReplace(v.Elem())
	case reflect.Interface:
		return shouldReplace(v.Elem())
	default:
		return false
	}
}
