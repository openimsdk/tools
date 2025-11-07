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
