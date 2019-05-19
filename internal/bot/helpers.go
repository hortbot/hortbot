package bot

import "strings"

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

func splitSpace(args string) (string, string) {
	a, b := splitFirstSep(args, " ")
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
