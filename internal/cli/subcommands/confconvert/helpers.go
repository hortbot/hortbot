package confconvert

import (
	"time"

	"github.com/go-sql-driver/mysql"
)

func unixMilli(milli int64) time.Time {
	return time.Unix(milli/1000, 1000000*(milli%1000))
}

func unixMilliPtr(milli *int64) time.Time {
	if milli == nil {
		return time.Time{}
	}
	return unixMilli(*milli)
}

func maybeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func init() {
	_ = mysql.SetLogger(noLog{})
}

type noLog struct{}

func (noLog) Print(v ...interface{}) {}
