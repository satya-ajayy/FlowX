package helpers

import "time"

func GetCurrentDateTime() string {
	utcTime := time.Now().UTC()
	return utcTime.Format("2006-01-02T15:04:05.999Z")
}

func GetNotEndedTime() string {
	zeroTime := time.Time{}
	return zeroTime.Format("2006-01-02T15:04:05.999Z")
}

func SecondsSince(start time.Time) int {
	return int(time.Since(start).Seconds())
}
