package bot

import (
	"context"
	"database/sql"
	"strings"
	"unicode"

	"github.com/jakebailey/irc"
	"go.opencensus.io/trace"
)

func splitFirstSep(s string, sep string) (string, string) {
	parts := strings.SplitN(s, sep, 2)

	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return parts[0], ""
	default:
		return parts[0], parts[1]
	}
}

func splitSpace(s string) (string, string) {
	a, b := splitFirstSep(s, " ")
	return a, strings.TrimSpace(b)
}

func parseBadges(badgeTag string) map[string]string {
	badges := strings.FieldsFunc(badgeTag, func(r rune) bool { return r == ',' })

	d := make(map[string]string, len(badges))

	for _, badge := range badges {
		k, v := splitFirstSep(badge, "/")
		d[k] = v
	}

	return d
}

func transact(ctx context.Context, db *sql.DB, fn func(context.Context, *sql.Tx) error) (err error) {
	ctx, span := trace.StartSpan(ctx, "transact")
	defer span.End()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	rollback := true

	defer func() {
		if rollback {
			if rerr := tx.Rollback(); err == nil && rerr != nil {
				err = rerr
			}
		}
	}()

	err = func() error {
		if _, err := tx.ExecContext(ctx, "SET LOCAL lock_timeout = '5s'"); err != nil {
			return err
		}
		return fn(ctx, tx)
	}()
	rollback = false

	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
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
