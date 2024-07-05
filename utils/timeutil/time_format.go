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
	"strconv"
	"time"

	"github.com/openimsdk/tools/errs"
)

const (
	TimeOffset = 8 * 3600  //8 hour offset
	HalfOffset = 12 * 3600 //Half-day hourly offset
)

// Get the current timestamp by Second
func GetCurrentTimestampBySecond() int64 {
	return time.Now().Unix()
}

// Convert timestamp to time.Time type
func UnixSecondToTime(second int64) time.Time {
	return time.Unix(second, 0)
}

// Convert nano timestamp to time.Time type
func UnixNanoSecondToTime(nanoSecond int64) time.Time {
	return time.Unix(0, nanoSecond)
}

// UnixMillSecondToTime convert millSecond to time.Time type
func UnixMillSecondToTime(millSecond int64) time.Time {
	return time.Unix(0, millSecond*1e6)
}

// Get the current timestamp by Nano
func GetCurrentTimestampByNano() int64 {
	return time.Now().UnixNano()
}

// Get the current timestamp by Mill
func GetCurrentTimestampByMill() int64 {
	return time.Now().UnixNano() / 1e6
}

// Get the timestamp at 0 o'clock of the day
func GetCurDayZeroTimestamp() int64 {
	timeStr := time.Now().Format("2006-01-02")
	t, _ := time.Parse("2006-01-02", timeStr)
	return t.Unix() - TimeOffset
}

// Get the timestamp at 12 o'clock on the day
func GetCurDayHalfTimestamp() int64 {
	return GetCurDayZeroTimestamp() + HalfOffset

}

// Get the formatted time at 0 o'clock of the day, the format is "2006-01-02_00-00-00"
func GetCurDayZeroTimeFormat() string {
	return time.Unix(GetCurDayZeroTimestamp(), 0).Format("2006-01-02_15-04-05")
}

// Get the formatted time at 12 o'clock of the day, the format is "2006-01-02_12-00-00"
func GetCurDayHalfTimeFormat() string {
	return time.Unix(GetCurDayZeroTimestamp()+HalfOffset, 0).Format("2006-01-02_15-04-05")
}

// GetTimeStampByFormat convert string to unix timestamp
func GetTimeStampByFormat(datetime string) string {
	timeLayout := "2006-01-02 15:04:05"
	loc, _ := time.LoadLocation("Local")
	tmp, _ := time.ParseInLocation(timeLayout, datetime, loc)
	timestamp := tmp.Unix()
	return strconv.FormatInt(timestamp, 10)
}

// TimeStringFormatTimeUnix convert string to unix timestamp
func TimeStringFormatTimeUnix(timeFormat string, timeSrc string) int64 {
	tm, _ := time.Parse(timeFormat, timeSrc)
	return tm.Unix()
}

// TimeStringToTime convert string to time.Time
func TimeStringToTime(timeString string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", timeString)
	return t, errs.WrapMsg(err, "timeStringToTime failed", "timeString", timeString)
}

// TimeToString convert time.Time to string
func TimeToString(t time.Time) string {
	return t.Format("2006-01-02")
}

func GetCurrentTimeFormatted() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// GetTimestampByTimezone get specific timestamp by timezone
func GetTimestampByTimezone(timezone string) (int64, error) {
	// set time zone
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return 0, errs.New("error loading location:", "error:", err)
	}
	// get current time
	currentTime := time.Now().In(location)
	// get timestamp
	timestamp := currentTime.Unix()
	return timestamp, nil
}

func DaysBetweenTimestamps(timezone string, timestamp int64) (int, error) {
	// set time zone
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return 0, errs.New("error loading location:", "error:", err)
	}
	// get current time
	now := time.Now().In(location)
	// timestamp to time
	givenTime := time.Unix(timestamp, 0)
	// calculate duration
	duration := now.Sub(givenTime)
	// change to days
	days := int(duration.Hours() / 24)
	return days, nil
}

// IsSameWeekday judge current day and specific day is the same of a week.
func IsSameWeekday(timezone string, timestamp int64) (bool, error) {
	// set time zone
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return false, errs.New("error loading location:", "error:", err)
	}
	// get current weekday
	currentWeekday := time.Now().In(location).Weekday()
	// change timestamp to weekday
	givenTime := time.Unix(timestamp, 0)
	givenWeekday := givenTime.Weekday()
	// compare two days
	return currentWeekday == givenWeekday, nil
}

func IsSameDayOfMonth(timezone string, timestamp int64) (bool, error) {
	// set time zone
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return false, errs.New("error loading location:", "error:", err)
	}
	// Get the current day of the month
	currentDay := time.Now().In(location).Day()
	// Convert the timestamp to time and get the day of the month
	givenDay := time.Unix(timestamp, 0).Day()
	// Compare the days
	return currentDay == givenDay, nil
}

func IsWeekday(timestamp int64) bool {
	// Convert the timestamp to time
	givenTime := time.Unix(timestamp, 0)
	// Get the day of the week
	weekday := givenTime.Weekday()
	// Check if the day is between Monday (1) and Friday (5)
	return weekday >= time.Monday && weekday <= time.Friday
}

func IsNthDayCycle(timezone string, startTimestamp int64, n int) (bool, error) {
	// set time zone
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return false, errs.New("error loading location:", "error:", err)
	}
	// Parse the start date
	startTime := time.Unix(startTimestamp, 0)
	if err != nil {
		return false, errs.New("invalid start date format:", "error:", err)
	}
	// Get the current time
	now := time.Now().In(location)
	// Calculate the difference in days between the current time and the start time
	diff := now.Sub(startTime).Hours() / 24
	// Check if the difference in days is a multiple of n
	return int(diff)%n == 0, nil
}

// IsNthWeekCycle checks if the current day is part of an N-week cycle starting from a given start timestamp.
func IsNthWeekCycle(timezone string, startTimestamp int64, n int) (bool, error) {
	// set time zone
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return false, errs.New("error loading location:", "error:", err)
	}

	// Get the current time
	now := time.Now().In(location)

	// Parse the start timestamp
	startTime := time.Unix(startTimestamp, 0)
	if err != nil {
		return false, errs.New("invalid start timestamp format:", "error:", err)
	}

	// Calculate the difference in days between the current time and the start time
	diff := now.Sub(startTime).Hours() / 24

	// Convert days to weeks
	weeks := int(diff) / 7

	// Check if the difference in weeks is a multiple of n
	return weeks%n == 0, nil
}

// IsNthMonthCycle checks if the current day is part of an N-month cycle starting from a given start timestamp.
func IsNthMonthCycle(timezone string, startTimestamp int64, n int) (bool, error) {
	// set time zone
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return false, errs.New("error loading location:", "error:", err)
	}

	// Get the current date
	now := time.Now().In(location)

	// Parse the start timestamp
	startTime := time.Unix(startTimestamp, 0)
	if err != nil {
		return false, errs.New("invalid start timestamp format:", "error:", err)
	}

	// Calculate the difference in months between the current time and the start time
	yearsDiff := now.Year() - startTime.Year()
	monthsDiff := int(now.Month()) - int(startTime.Month())
	totalMonths := yearsDiff*12 + monthsDiff

	// Check if the difference in months is a multiple of n
	return totalMonths%n == 0, nil
}
