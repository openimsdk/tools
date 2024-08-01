package mw

import (
	"github.com/openimsdk/tools/utils/datautil"
	"reflect"
)

// ReplaceNil initialization nil values.
// e.g. Slice will be initialized as [],Map/interface will be initialized as {}
func ReplaceNil(data any) {
	v := reflect.ValueOf(data)

	replaceNil(v)
}

func replaceNil(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			// Handle multi-level pointers
			elemKind := v.Type().Elem().Kind()
			if elemKind == reflect.Pointer {
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
			fieldType := v.Type().Field(i)

			// Check if the field is exported
			if fieldType.IsExported() {
				replaceNil(field)
			}
		}
	case reflect.Interface:
		if !v.IsNil() && !shouldReplace(v) {
			// If the interface is already initialized, recursively replace the internal nils
			replaceNil(v.Elem())
		} else {
			// If the interface is not initialized, the struct will be initialized as {}
			realType := getRealType(v.Interface())
			if realType == nil {
				// Invalid type
				return
			}
			switch realType.Kind() {
			case reflect.Slice:
				v.Set(reflect.MakeSlice(realType, 0, 0))
			case reflect.Map:
				v.Set(reflect.MakeMap(realType))
			case reflect.Struct:
				v.Set(reflect.New(reflect.TypeOf(struct{}{})))
			default:
			}
		}
	default:
		return
	}
}

// getRealType determines the underlying type.
func getRealType(data any) reflect.Type {
	t := reflect.TypeOf(data)
	if t == nil {
		return t
	}
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	return t
}

// shouldReplace determines whether internal replacement is needed,
// checks if the underlying component has already been initialized.
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
