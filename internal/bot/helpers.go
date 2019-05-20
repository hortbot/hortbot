package bot

import (
	"database/sql"
	"strings"
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

func transact(db *sql.DB, fn func(*sql.Tx) error) (err error) {
	var tx *sql.Tx
	tx, err = db.Begin()
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

	err = fn(tx)
	rollback = false

	if err != nil {
		tx.Rollback() //nolint:errcheck
		return err
	}

	return tx.Commit()
}
