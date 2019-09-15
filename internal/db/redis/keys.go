package redis

import "strings"

type keyStr string

func (k keyStr) is(v string) keyPair {
	return keyPair{
		name:  string(k),
		value: v,
	}
}

type keyPair struct {
	name  string
	value string
}

func checkKey(s string) {
	if strings.ContainsRune(s, ':') {
		panic("key contains colon: " + s)
	}
}

func buildKey(pairs ...keyPair) string {
	if len(pairs) == 0 {
		panic("no key specified")
	}

	size := 2 * len(pairs)

	for _, p := range pairs {
		checkKey(p.name)
		checkKey(p.value)
		size += len(p.name) + len(p.value)
	}

	var b strings.Builder
	b.Grow(size)

	for i, p := range pairs {
		if i != 0 {
			b.WriteByte(':')
		}

		b.WriteString(p.name)
		b.WriteByte(':')
		b.WriteString(p.value)
	}

	return b.String()
}
