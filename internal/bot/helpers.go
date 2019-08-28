package bot

import (
	"context"
	"strings"
	"unicode"

	"github.com/hortbot/hortbot/internal/db/dbtrace"
	"github.com/opentracing/opentracing-go"
	"github.com/volatiletech/sqlboiler/boil"
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

func transact(ctx context.Context, db boil.Beginner, fn func(context.Context, boil.ContextExecutor) error) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "transact")
	defer span.Finish()

	dtx, err := db.Begin()
	if err != nil {
		return err
	}

	tx := dbtrace.Tx(dtx)

	rollback := true

	defer func() {
		if rollback {
			if rerr := tx.Rollback(); err == nil && rerr != nil {
				err = rerr
			}
		}
	}()

	err = fn(ctx, tx)
	rollback = false

	if err != nil {
		tx.Rollback() //nolint:errcheck
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
