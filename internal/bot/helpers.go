package bot

import (
	"context"
	"strconv"
	"strings"
	"unicode"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/stringsx"
)

func splitSpace(s string) (string, string) {
	a, b := stringsx.SplitByte(s, ' ')
	return a, strings.TrimSpace(b)
}

func stringSliceIndex(strs []string, s string) (int, bool) {
	for i, v := range strs {
		if s == v {
			return i, true
		}
	}
	return -1, false
}

func cleanCommandName(s string) string {
	m := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return r
		}
		return -1
	}, s)

	// In the common case, Map won't modify the string, and neither will
	// ToLower, so this is faster than making Map do everything.
	return strings.ToLower(m)
}

func writeBool(b *strings.Builder, v bool) {
	if v {
		b.WriteString("true")
	} else {
		b.WriteString("false")
	}
}

func cleanUsername(user string) string {
	user = strings.TrimPrefix(user, "@")
	return strings.ToLower(user)
}

func parseInt32(s string) (int32, error) {
	value, err := strconv.ParseInt(s, 10, 32)
	return int32(value), err
}

func pluralInt[T ~int | ~int32 | ~int64](n T, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func pgLock(ctx context.Context, queries *dbsql.Queries, twitchID int64) error {
	return queries.AcquireTwitchAdvisoryLock(ctx, twitchID) //nolint:wrapcheck
}
