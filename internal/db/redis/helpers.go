package redis

import (
	"strings"
)

func buildKey(parts ...string) string {
	if len(parts) == 0 {
		panic("no key specified")
	}

	for _, p := range parts {
		if strings.ContainsRune(p, ':') {
			panic("key contains colon")
		}
	}

	return strings.Join(parts, ":")
}
