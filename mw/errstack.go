package mw

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

func FormatError(err error) error {
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}
	if e, ok := err.(stackTracer); ok {
		st := e.StackTrace()
		var sb strings.Builder
		sb.WriteString("Error: ")
		sb.WriteString(err.Error())
		sb.WriteString(" | Error trace: ")

		var callPath []string
		for _, f := range st {
			pc := uintptr(f) - 1
			fn := runtime.FuncForPC(pc)
			if fn == nil {
				continue
			}
			if strings.Contains(fn.Name(), "runtime.") {
				continue
			}
			file, line := fn.FileLine(pc)
			funcName := simplifyFuncName(fn.Name())
			callPath = append(callPath, fmt.Sprintf("%s (%s:%d)", funcName, file, line))
		}
		for i := len(callPath) - 1; i >= 0; i-- {
			if i != len(callPath)-1 {
				sb.WriteString(" -> ")
			}
			sb.WriteString(callPath[i])
		}
		return errors.New(sb.String())
	}
	return err
}
func simplifyFuncName(fullFuncName string) string {
	parts := strings.Split(fullFuncName, "/")
	lastPart := parts[len(parts)-1]
	parts = strings.Split(lastPart, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return lastPart
}
