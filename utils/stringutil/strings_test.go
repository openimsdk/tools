package stringutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntToString(t *testing.T) {
	assert.Equal(t, "123", IntToString(123))
}

func TestStringToInt(t *testing.T) {
	assert.Equal(t, 123, StringToInt("123"))
}

func TestStringToInt64(t *testing.T) {
	assert.Equal(t, int64(123), StringToInt64("123"))
}

func TestStringToInt32(t *testing.T) {
	assert.Equal(t, int32(123), StringToInt32("123"))
}

func TestInt32ToString(t *testing.T) {
	assert.Equal(t, "123", Int32ToString(123))
}

func TestUint32ToString(t *testing.T) {
	assert.Equal(t, "123", Uint32ToString(123))
}

func TestIsContain(t *testing.T) {
	list := []string{"apple", "banana", "cherry"}
	assert.True(t, IsContain("banana", list))
	assert.False(t, IsContain("date", list))
}

func TestIsContainInt32(t *testing.T) {
	list := []int32{1, 2, 3}
	assert.True(t, IsContainInt32(2, list))
	assert.False(t, IsContainInt32(4, list))
}

func TestIsContainInt(t *testing.T) {
	list := []int{1, 2, 3}
	assert.True(t, IsContainInt(2, list))
	assert.False(t, IsContainInt(4, list))
}

func TestRemoveDuplicateElement(t *testing.T) {
	idList := []string{"a", "b", "a", "c", "b"}
	expected := []string{"a", "b", "c"}
	result := RemoveDuplicateElement(idList)
	assert.ElementsMatch(t, expected, result)
}

func TestIsDuplicateStringSlice(t *testing.T) {
	assert.False(t, IsDuplicateStringSlice([]string{"a", "b", "c"}))
	assert.True(t, IsDuplicateStringSlice([]string{"a", "b", "a"}))
}

func TestFormatString(t *testing.T) {
	cases := []struct {
		name      string
		text      string
		length    int
		alignLeft bool
		want      string
	}{
		{"LeftAlignShort", "hello", 10, true, "hello     "},
		{"RightAlignShort", "hello", 10, false, "     hello"},
		{"ExactLength", "hello", 5, true, "hello"},
		{"TruncateLong", "hello world", 5, true, "hello"},
		{"LeftAlignEmpty", "", 5, true, "     "},
		{"RightAlignEmpty", "", 5, false, "     "},
		{"NoLength", "hello", 0, true, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatString(tc.text, tc.length, tc.alignLeft)
			if got != tc.want {
				t.Errorf("FormatString(%q, %d, %t) = %q; want %q", tc.text, tc.length, tc.alignLeft, got, tc.want)
			}
		})
	}
}

func TestCamelCaseToSpaceSeparated(t *testing.T) {
	inputs := []string{
		"HelloWorld,GoGo",
		"hello world, go go",
	}
	for _, input := range inputs {
		r := CamelCaseToSpaceSeparated(input)
		fmt.Println(r)
	}
}
