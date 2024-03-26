package idutil

import (
	"github.com/openimsdk/tools/utils/encrypt"
	"github.com/openimsdk/tools/utils/stringutil"
	"github.com/openimsdk/tools/utils/timeutil"
	"math/rand"
)

func GetMsgIDByMD5(sendID string) string {
	t := stringutil.Int64ToString(timeutil.GetCurrentTimestampByNano())
	return encrypt.Md5(t + sendID + stringutil.Int64ToString(rand.Int63n(timeutil.GetCurrentTimestampByNano())))
}
