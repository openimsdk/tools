package checker_test

import (
	"testing"

	"github.com/openimsdk/tools/checker"
	"github.com/openimsdk/tools/errs"
	"github.com/stretchr/testify/assert"
)

type mockChecker struct {
	err error
}

func (m mockChecker) Check() error {
	return m.err
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		arg       any
		wantError error
	}{
		{
			name:      "non-checker argument",
			arg:       "non-checker",
			wantError: nil,
		},
		{
			name:      "checker with no error",
			arg:       mockChecker{nil},
			wantError: nil,
		},
		{
			name:      "checker with generic error",
			arg:       mockChecker{errs.New("generic error")},
			wantError: errs.ErrArgs,
		},
		{
			name:      "checker with CodeError",
			arg:       mockChecker{errs.NewCodeError(400, "bad request")},
			wantError: errs.NewCodeError(400, "bad request"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.Validate(tt.arg)
			if tt.wantError != nil {
				assert.ErrorIs(t, err, tt.wantError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
