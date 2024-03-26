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

package timeutil

import (
	"testing"
	"time"
)

func TestGetCurrentTimestampBySecond(t *testing.T) {
	now := time.Now().Unix()
	got := GetCurrentTimestampBySecond()
	if got < now {
		t.Errorf("GetCurrentTimestampBySecond() = %v, want at least %v", got, now)
	}
}

func TestUnixSecondToTime(t *testing.T) {
	now := time.Now().Unix()
	got := UnixSecondToTime(now)
	if got.Unix() != now {
		t.Errorf("UnixSecondToTime(%v) = %v, want %v", now, got.Unix(), now)
	}
}

func TestUnixNanoSecondToTime(t *testing.T) {
	now := time.Now().UnixNano()
	got := UnixNanoSecondToTime(now)
	if got.UnixNano() != now {
		t.Errorf("UnixNanoSecondToTime(%v) = %v, want %v", now, got.UnixNano(), now)
	}
}

func TestUnixMillSecondToTime(t *testing.T) {
	now := time.Now().UnixNano() / 1e6
	got := UnixMillSecondToTime(now)
	if got.UnixNano()/1e6 != now {
		t.Errorf("UnixMillSecondToTime(%v) = %v, want %v", now, got.UnixNano()/1e6, now)
	}
}

func TestGetCurrentTimestampByNano(t *testing.T) {
	now := time.Now().UnixNano()
	got := GetCurrentTimestampByNano()
	if got < now {
		t.Errorf("GetCurrentTimestampByNano() = %v, want at least %v", got, now)
	}
}

func TestGetCurrentTimestampByMill(t *testing.T) {
	now := time.Now().UnixNano() / 1e6
	got := GetCurrentTimestampByMill()
	if got < now {
		t.Errorf("GetCurrentTimestampByMill() = %v, want at least %v", got, now)
	}
}

func TestGetCurDayHalfTimestamp(t *testing.T) {
	expected := GetCurDayZeroTimestamp() + HalfOffset
	got := GetCurDayHalfTimestamp()
	if got != expected {
		t.Errorf("GetCurDayHalfTimestamp() = %v, want %v", got, expected)
	}
}

func TestGetCurDayZeroTimeFormat(t *testing.T) {
	// This test may need adjustment based on the local time zone of the testing environment.
	expected := time.Unix(GetCurDayZeroTimestamp(), 0).Format("2006-01-02_15-04-05")
	got := GetCurDayZeroTimeFormat()
	if got != expected {
		t.Errorf("GetCurDayZeroTimeFormat() = %v, want %v", got, expected)
	}
}

func TestGetCurDayHalfTimeFormat(t *testing.T) {
	// This test may need adjustment based on the local time zone of the testing environment.
	expected := time.Unix(GetCurDayZeroTimestamp()+HalfOffset, 0).Format("2006-01-02_15-04-05")
	got := GetCurDayHalfTimeFormat()
	if got != expected {
		t.Errorf("GetCurDayHalfTimeFormat() = %v, want %v", got, expected)
	}
}
