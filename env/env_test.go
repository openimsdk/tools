package env

import (
	"os"
	"strconv"
	"testing"
)

func TestGetString(t *testing.T) {
	const expected = "foo"

	key := "STRING_SET_VAR"
	os.Setenv(key, expected)
	if e, a := expected, GetString(key, "~"+expected); e != a {
		t.Fatalf("expected %#v==%#v", e, a)
	}

	key = "STRING_UNSET_VAR"
	if e, a := expected, GetString(key, expected); e != a {
		t.Fatalf("expected %#v==%#v", e, a)
	}
}

func TestGetInt(t *testing.T) {
	const expected = 1
	const defaultValue = 2

	key := "INT_SET_VAR"
	os.Setenv(key, strconv.Itoa(expected))
	returnVal, _ := GetInt(key, defaultValue)
	if e, a := expected, returnVal; e != a {
		t.Fatalf("expected %#v==%#v", e, a)
	}

	key = "INT_UNSET_VAR"
	returnVal, _ = GetInt(key, defaultValue)
	if e, a := defaultValue, returnVal; e != a {
		t.Fatalf("expected %#v==%#v", e, a)
	}

	key = "INT_SET_VAR"
	os.Setenv(key, "not-an-int")
	returnVal, err := GetInt(key, defaultValue)
	if e, a := defaultValue, returnVal; e != a {
		t.Fatalf("expected %#v==%#v", e, a)
	}
	if err == nil {
		t.Error("expected error")
	}
}

func TestGetFloat64(t *testing.T) {
	const expected = 1.0
	const defaultValue = 2.0

	key := "FLOAT_SET_VAR"
	os.Setenv(key, "1.0")
	returnVal, _ := GetFloat64(key, defaultValue)
	if e, a := expected, returnVal; e != a {
		t.Fatalf("expected %#v==%#v", e, a)
	}

	key = "FLOAT_UNSET_VAR"
	returnVal, _ = GetFloat64(key, defaultValue)
	if e, a := defaultValue, returnVal; e != a {
		t.Fatalf("expected %#v==%#v", e, a)
	}

	key = "FLOAT_SET_VAR"
	os.Setenv(key, "not-a-float")
	returnVal, err := GetFloat64(key, defaultValue)
	if e, a := defaultValue, returnVal; e != a {
		t.Fatalf("expected %#v==%#v", e, a)
	}
	if err == nil || err.Error() != "strconv.ParseFloat: parsing \"not-a-float\": invalid syntax" {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestGetBool(t *testing.T) {
	const expected = true
	const defaultValue = false

	key := "BOOL_SET_VAR"
	os.Setenv(key, "true")
	returnVal, _ := GetBool(key, defaultValue)
	if e, a := expected, returnVal; e != a {
		t.Fatalf("expected %#v==%#v", e, a)
	}

	key = "BOOL_UNSET_VAR"
	returnVal, _ = GetBool(key, defaultValue)
	if e, a := defaultValue, returnVal; e != a {
		t.Fatalf("expected %#v==%#v", e, a)
	}

	key = "BOOL_SET_VAR"
	os.Setenv(key, "not-a-bool")
	returnVal, err := GetBool(key, defaultValue)
	if e, a := defaultValue, returnVal; e != a {
		t.Fatalf("expected %#v==%#v", e, a)
	}
	if err == nil || err.Error() != "strconv.ParseBool: parsing \"not-a-bool\": invalid syntax" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
