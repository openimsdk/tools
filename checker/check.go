package checker

import "github.com/openimsdk/tools/errs"

type Checker interface {
	Check() error
}

func Validate(args any) error {
	if checker, ok := args.(Checker); ok {
		if err := checker.Check(); err != nil {
			if _, ok := errs.Unwrap(err).(errs.CodeError); ok {
				return err
			}
			return errs.ErrArgs.WrapMsg(err.Error())
		}
	}
	return nil
}
