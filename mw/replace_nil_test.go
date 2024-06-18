package mw

import (
	"fmt"
	"reflect"
	"testing"
)

type A struct {
	B *B
	C []int
	D map[string]string
	E interface{}
	F *int
}

type B struct {
	D *C
	E []int
}

type C struct {
	F int
}

func TestReplaceNil(t *testing.T) {
	var a ***A
	ReplaceNil(&a)
	fmt.Println("struct a:")
	printStruct(reflect.ValueOf(a), "")
	//struct a:
	//(pointer)
	//  (pointer)
	//    (pointer)
	//B:         <nil>
	//C: []
	//D: map[]
	//E: <nil>
	//F:         <nil>

	b := &A{}
	ReplaceNil(b)
	fmt.Println("struct: b")
	printStruct(reflect.ValueOf(b), "")
	//struct: b
	//(pointer)
	//B:     (pointer)
	//    D:         <nil>
	//    E: []
	//C: []
	//D: map[]
	//E: <nil>
	//F:     (pointer)
}

func printStruct(v reflect.Value, indent string) {
	originalIndent := indent

	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			fmt.Printf("%s<nil>\n", indent)
			return
		}
		fmt.Printf("%s(pointer)\n", indent)
		v = v.Elem()
		indent += "  "
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := v.Type().Field(i).Name

		switch field.Kind() {
		case reflect.Ptr:
			fmt.Printf("%s%s: ", originalIndent, fieldName)
			printStruct(field, indent+"  ")
		case reflect.Slice:
			if field.IsNil() {
				fmt.Printf("%s%s: []\n", originalIndent, fieldName)
			} else {
				fmt.Printf("%s%s: %v\n", originalIndent, fieldName, field.Interface())
			}
		case reflect.Map:
			if field.IsNil() {
				fmt.Printf("%s%s: map[]\n", originalIndent, fieldName)
			} else {
				fmt.Printf("%s%s: %v\n", originalIndent, fieldName, field.Interface())
			}
		case reflect.Interface:
			if field.IsNil() {
				fmt.Printf("%s%s: <nil>\n", originalIndent, fieldName)
			} else {
				fmt.Printf("%s%s: (interface)\n", originalIndent, fieldName)
				printStruct(field.Elem(), indent+"  ")
			}
		case reflect.Struct:
			fmt.Printf("%s%s: (struct)\n", originalIndent, fieldName)
			printStruct(field, indent+"  ")
		default:
			fmt.Printf("%s%s: %v\n", originalIndent, fieldName, field.Interface())
		}
	}
}
