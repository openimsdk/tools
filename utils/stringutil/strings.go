// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stringutil

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

func IntToString(i int) string {
	return strconv.FormatInt(int64(i), 10)
}

func StringToInt(i string) int {
	j, _ := strconv.Atoi(i)
	return j
}
func StringToInt64(i string) int64 {
	j, _ := strconv.ParseInt(i, 10, 64)
	return j
}
func StringToInt32(i string) int32 {
	j, _ := strconv.ParseInt(i, 10, 64)
	return int32(j)
}
func Int32ToString(i int32) string {
	return strconv.FormatInt(int64(i), 10)
}

func Uint32ToString(i uint32) string {
	return strconv.FormatInt(int64(i), 10)
}

// judge a string whether in the  string list
func IsContain(target string, List []string) bool {
	for _, element := range List {

		if target == element {
			return true
		}
	}
	return false
}
func IsContainInt32(target int32, List []int32) bool {
	for _, element := range List {
		if target == element {
			return true
		}
	}
	return false
}
func IsContainInt(target int, List []int) bool {
	for _, element := range List {
		if target == element {
			return true
		}
	}
	return false
}
func InterfaceArrayToStringArray(data []any) (i []string) {
	for _, param := range data {
		i = append(i, param.(string))
	}
	return i
}

func StructToJsonBytes(param any) []byte {
	dataType, _ := json.Marshal(param)
	return dataType
}

func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

func RemoveDuplicateElement(idList []string) []string {
	result := make([]string, 0, len(idList))
	temp := map[string]struct{}{}
	for _, item := range idList {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func RemoveDuplicate[T comparable](arr []T) []T {
	result := make([]T, 0, len(arr))
	temp := map[T]struct{}{}
	for _, item := range arr {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func IsDuplicateStringSlice(arr []string) bool {
	t := make(map[string]struct{})
	for _, s := range arr {
		if _, ok := t[s]; ok {
			return true
		}
		t[s] = struct{}{}
	}
	return false
}

func GetSelfFuncName() string {
	pc, _, _, _ := runtime.Caller(1)
	return cleanUpFuncName(runtime.FuncForPC(pc).Name())
}

func GetFuncName(skips ...int) string {
	skip := 1
	if len(skips) > 0 {
		skip = skips[0] + 1
	}
	pc, _, _, _ := runtime.Caller(skip)
	return cleanUpFuncName(runtime.FuncForPC(pc).Name())
}

func cleanUpFuncName(funcName string) string {
	end := strings.LastIndex(funcName, ".")
	if end == -1 {
		return ""
	}
	return funcName[end+1:]
}

// Get the intersection of two slices
func IntersectString(slice1, slice2 []string) []string {
	m := make(map[string]bool)
	n := make([]string, 0)
	for _, v := range slice1 {
		m[v] = true
	}
	for _, v := range slice2 {
		flag, _ := m[v]
		if flag {
			n = append(n, v)
		}
	}
	return n
}

// Get the diff of two slices
func DifferenceString(slice1, slice2 []string) []string {
	m := make(map[string]bool)
	n := make([]string, 0)
	inter := IntersectString(slice1, slice2)
	for _, v := range inter {
		m[v] = true
	}
	for _, v := range slice1 {
		if !m[v] {
			n = append(n, v)
		}
	}

	for _, v := range slice2 {
		if !m[v] {
			n = append(n, v)
		}
	}
	return n
}

// Get the intersection of two slices
func Intersect(slice1, slice2 []int64) []int64 {
	m := make(map[int64]bool)
	n := make([]int64, 0)
	for _, v := range slice1 {
		m[v] = true
	}
	for _, v := range slice2 {
		flag, _ := m[v]
		if flag {
			n = append(n, v)
		}
	}
	return n
}

// Get the diff of two slices
func Difference(slice1, slice2 []int64) []int64 {
	m := make(map[int64]bool)
	n := make([]int64, 0)
	inter := Intersect(slice1, slice2)
	for _, v := range inter {
		m[v] = true
	}
	for _, v := range slice1 {
		if !m[v] {
			n = append(n, v)
		}
	}

	for _, v := range slice2 {
		if !m[v] {
			n = append(n, v)
		}
	}
	return n
}

func GetHashCode(s string) uint32 {
	return crc32.ChecksumIEEE([]byte(s))
}

// FormatString formats a string with a specified length and alignment.
// `text` is the input string to format.
// `length` is the desired length of the output string.
// `alignLeft` specifies whether the string should be left-aligned (true) or right-aligned (false).
func FormatString(text string, length int, alignLeft bool) string {
	if len(text) > length {
		// Truncate the string if it's longer than the desired length
		return text[:length]
	}

	// Create a format string based on alignment preference
	var formatStr string
	if alignLeft {
		formatStr = fmt.Sprintf("%%-%ds", length) // Left align
	} else {
		formatStr = fmt.Sprintf("%%%ds", length) // Right align
	}

	// Use the format string to format the text
	return fmt.Sprintf(formatStr, text)
}

// CamelCaseToSpaceSeparated converts a camelCase string to a space-separated format
func CamelCaseToSpaceSeparated(input string) string {
	var result []rune
	for i, r := range input {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, ' ')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// UpperFirst upper the first letter of the input string
func UpperFirst(input string) string {
	if len(input) == 0 {
		return input
	}
	runes := []rune(input)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// LowerFirst lower the first letter of the input string
func LowerFirst(input string) string {
	if len(input) == 0 {
		return input
	}
	runes := []rune(input)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func IsAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsValidEmail checks if the input string is a valid email address.
func IsValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}
