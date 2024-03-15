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

package errs

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type CodeError interface {
	Code() int
	Msg() string
	Detail() string
	WithDetail(detail string) CodeError
	// Is 判断是否是某个错误, loose为false时, 只有错误码相同就认为是同一个错误, 默认为true
	Is(err error, loose ...bool) bool
	Wrap(msg string, kv ...any) error
	error
}

func NewCodeError(code int, msg string) CodeError {
	return &codeError{
		code: code,
		msg:  msg,
	}
}

type codeError struct {
	code   int
	msg    string
	detail string
}

func (e *codeError) Code() int {
	return e.code
}

func (e *codeError) Msg() string {
	return e.msg
}

func (e *codeError) Detail() string {
	return e.detail
}

func (e *codeError) WithDetail(detail string) CodeError {
	var d string
	if e.detail == "" {
		d = detail
	} else {
		d = e.detail + ", " + detail
	}
	return &codeError{
		code:   e.code,
		msg:    e.msg,
		detail: d,
	}
}

func (e *codeError) Wrap(msg string, kv ...any) error {
	return Wrap(e, msg, kv...)
}

func (e *codeError) Is(err error, loose ...bool) bool {
	if err == nil {
		return false
	}
	var allowSubclasses bool
	if len(loose) == 0 {
		allowSubclasses = true
	} else {
		allowSubclasses = loose[0]
	}
	codeErr, ok := Unwrap(err).(CodeError)
	if ok {
		if allowSubclasses {
			return Relation.Is(e.code, codeErr.Code())
		} else {
			return codeErr.Code() == e.code
		}
	}
	return false
}

func (e *codeError) Error() string {
	v := make([]string, 0, 3)
	v = append(v, strconv.Itoa(e.code), e.msg)
	if e.detail != "" {
		v = append(v, e.detail)
	}
	return strings.Join(v, " ")
}

func Unwrap(err error) error {
	for err != nil {
		unwrap, ok := err.(interface {
			Unwrap() error
		})
		if !ok {
			break
		}
		err = unwrap.Unwrap()
	}
	return err
}

func Wrap(err error, msg string, kv ...any) error {
	if len(kv) == 0 {
		if len(msg) == 0 {
			return errors.WithStack(err)
		} else {
			return errors.WithMessage(err, msg)
		}
	}
	var buf bytes.Buffer
	if len(msg) > 0 {
		buf.WriteString(msg)
		buf.WriteString(" ")
	}
	for i := 0; i < len(kv); i += 2 {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(toString(kv[i]))
		buf.WriteString("=")
		buf.WriteString(toString(kv[i+1]))
	}
	return errors.WithMessage(err, buf.String())
}

func toString(v any) string {
	const nilStr = "<nil>"
	if v == nil {
		return nilStr
	}
	switch w := v.(type) {
	case string:
		return w
	case []byte:
		return string(w)
	case []rune:
		return string(w)
	case int:
		return strconv.Itoa(w)
	case int8:
		return strconv.FormatInt(int64(w), 10)
	case int16:
		return strconv.FormatInt(int64(w), 10)
	case int32:
		return strconv.FormatInt(int64(w), 10)
	case int64:
		return strconv.FormatInt(w, 10)
	case uint:
		return strconv.FormatUint(uint64(w), 10)
	case uint8:
		return strconv.FormatUint(uint64(w), 10)
	case uint16:
		return strconv.FormatUint(uint64(w), 10)
	case uint32:
		return strconv.FormatUint(uint64(w), 10)
	case uint64:
		return strconv.FormatUint(w, 10)
	case float32:
		return strconv.FormatFloat(float64(w), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(w, 'f', -1, 64)
	case error:
		if w == nil {
			return nilStr
		}
		return w.Error()
	case fmt.Stringer:
		return w.String()
	case *string:
		if w == nil {
			return nilStr
		}
		return *w
	case *[]byte:
		if w == nil {
			return nilStr
		}
		return string(*w)
	case *[]rune:
		if w == nil {
			return nilStr
		}
		return string(*w)
	case *int:
		if w == nil {
			return nilStr
		}
		return strconv.Itoa(*w)
	case *int8:
		if w == nil {
			return nilStr
		}
		return strconv.FormatInt(int64(*w), 10)
	case *int16:
		if w == nil {
			return nilStr
		}
		return strconv.FormatInt(int64(*w), 10)
	case *int32:
		if w == nil {
			return nilStr
		}
		return strconv.FormatInt(int64(*w), 10)
	case *int64:
		if w == nil {
			return nilStr
		}
		return strconv.FormatInt(*w, 10)
	case *uint:
		if w == nil {
			return nilStr
		}
		return strconv.FormatUint(uint64(*w), 10)
	case *uint8:
		if w == nil {
			return nilStr
		}
		return strconv.FormatUint(uint64(*w), 10)
	case *uint16:
		if w == nil {
			return nilStr
		}
		return strconv.FormatUint(uint64(*w), 10)
	case *uint32:
		if w == nil {
			return nilStr
		}
		return strconv.FormatUint(uint64(*w), 10)
	case *uint64:
		if w == nil {
			return nilStr
		}
		return strconv.FormatUint(*w, 10)
	case *float32:
		if w == nil {
			return nilStr
		}
		return strconv.FormatFloat(float64(*w), 'f', -1, 32)
	case *float64:
		if w == nil {
			return nilStr
		}
		return strconv.FormatFloat(*w, 'f', -1, 64)
	case *error:
		if w == nil {
			return nilStr
		}
		return (*w).Error()
	case *fmt.Stringer:
		if w == nil {
			return nilStr
		}
		return (*w).String()
	default:
		return fmt.Sprintf("%+v", w)
	}
}
