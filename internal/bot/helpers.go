package bot

import (
	"context"
	"database/sql"
	"strings"
	"unicode"

	"github.com/hortbot/hortbot/internal/pkg/stringsx"
	"github.com/jakebailey/irc"
)

func splitSpace(s string) (string, string) {
	a, b := stringsx.SplitByte(s, ' ')
	return a, strings.TrimSpace(b)
}

func parseBadges(badgeTag string) map[string]string {
	badges := strings.FieldsFunc(badgeTag, func(r rune) bool { return r == ',' })

	d := make(map[string]string, len(badges))

	for _, badge := range badges {
		k, v := stringsx.SplitByte(badge, '/')
		d[k] = v
	}

	return d
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

func readMessage(m *irc.Message) (message string, me bool) {
	message = m.Trailing

	if c, a, ok := irc.ParseCTCP(message); ok {
		if c != "ACTION" {
			return "", false
		}

		message = a
		me = true
	}

	return strings.TrimSpace(message), me
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
	return err
}

func ptrTo[T any](v T) *T {
	return &v
}
