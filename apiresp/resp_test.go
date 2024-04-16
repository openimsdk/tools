package apiresp

import (
	"github.com/openimsdk/protocol/auth"
	"github.com/openimsdk/tools/utils/jsonutil"
	"math"
	"testing"
)

func TestName(t *testing.T) {
	resp := &ApiResponse{
		ErrCode: 1234,
		ErrMsg:  "test",
		ErrDlt:  "4567",
		Data: &auth.UserTokenResp{
			Token:             "1234567",
			ExpireTimeSeconds: math.MaxInt64,
		},
	}
	data, err := resp.MarshalJSON()
	if err != nil {
		panic(err)
	}
	t.Log(string(data))

	var rReso ApiResponse
	rReso.Data = &auth.UserTokenResp{}

	if err := jsonutil.JsonUnmarshal(data, &rReso); err != nil {
		panic(err)
	}

	t.Logf("%+v\n", rReso)

}
