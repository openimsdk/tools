package errs

//import (
//	"fmt"
//	"runtime"
//	"strings"
//)
//
//func GetPanicStack() string {
//	var pcs [32]uintptr
//	n := runtime.Callers(0, pcs[:])
//	frames := runtime.CallersFrames(pcs[:n])
//
//	var (
//		sb              strings.Builder
//		frame           runtime.Frame
//		beginPanicStack = false
//		more            = true
//		begin           = true
//	)
//
//	for {
//		if !more {
//			break
//		}
//		frame, more = frames.Next()
//		if !beginPanicStack && !strings.Contains(frame.Function, "gopanic") {
//
//			continue
//		} else {
//			beginPanicStack = true
//		}
//
//		if strings.HasPrefix(frame.Function, "runtime.") {
//			continue
//		}
//
//		if begin {
//			begin = false
//		} else {
//			sb.WriteString(" -> ")
//		}
//		funcNameParts := strings.Split(frame.Function, ".")
//		s := fmt.Sprintf("%s (%s:%d)", funcNameParts[len(funcNameParts)-1], frame.File, frame.Line)
//		sb.WriteString(s)
//	}
//	return sb.String()
//}
