package confconvert

import (
	"time"

	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

type noTrace struct{}

func (noTrace) Enabled(zapcore.Level) bool { return false }

func init() {
	_ = mysql.SetLogger(noLog{})
}

type noLog struct{}

func (noLog) Print(v ...interface{}) {}

type plainError struct {
	e error
}

func (pe plainError) Error() string {
	return pe.e.Error()
}

func PlainError(err error) zap.Field {
	return zap.Error(plainError{e: err})
}
