// Copyright © 2024 OpenIM open source community. All rights reserved.
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

package validation

import (
	"reflect"

	"github.com/openimsdk/tools/errs"
)

// Validator defines the interface for configuration validators.
type Validator interface {
	Validate(config any) error
}

// SimpleValidator is a basic implementation of the Validator interface,
// which checks if fields in the configuration satisfy basic non-zero value requirements using reflection.
type SimpleValidator struct{}

// NewSimpleValidator creates and returns an instance of SimpleValidator.
func NewSimpleValidator() *SimpleValidator {
	return &SimpleValidator{}
}

// Validate checks if all fields in the given configuration object are set (i.e., non-zero values).
// This is a very basic implementation and may need to be extended based on specific application requirements.
func (v *SimpleValidator) Validate(config any) error {
	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Validation is performed only when config is a struct
	if val.Kind() == reflect.Struct {
		for i := 0; i < val.NumField(); i++ {
			// Check if the field has a zero value
			if isZeroOfUnderlyingType(val.Field(i).Interface()) {
				// If it has a zero value, return an error indicating which field is unset
				return errs.New("validation failed: field " + val.Type().Field(i).Name + " is zero value").Wrap()
			}
		}
	} else {
		return errs.New("validation failed: config must be a struct or a pointer to struct").Wrap()
	}

	return nil
}

// isZeroOfUnderlyingType checks if a value is the zero value of its type.
func isZeroOfUnderlyingType(x any) bool {
	return x == reflect.Zero(reflect.TypeOf(x)).Interface()
}
