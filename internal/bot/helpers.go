package bot

import (
	"context"
	"database/sql"
	"strings"
	"unicode"

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

func pluralInt(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func pluralInt64(n int64, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func pgLock(ctx context.Context, tx *sql.Tx, twitchID int64) error {
	_, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", twitchID)
	return err //nolint:wrapcheck
}

func ptrTo[T any](v T) *T {
	return &v
}
