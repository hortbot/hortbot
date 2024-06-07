package wqueue

import (
	"strconv"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"gotest.tools/v3/assert"
)

func TestOrderedSet(t *testing.T) {
	t.Parallel()
	set := newOrderedSet[string, *subQueue]()

	k, v := set.next()
	assert.Equal(t, k, "")
	assert.Equal(t, v, (*subQueue)(nil))

	pairs := make([]pair[string, *subQueue], 100)

	for i := range pairs {
		p := pair[string, *subQueue]{
			key:   strconv.Itoa(i),
			value: &subQueue{},
		}
		pairs[i] = p
		set.add(p.key, p.value)
	}

	for _, p := range pairs {
		assert.Equal(t, set.find(p.key), p.value)
	}

	assert.Equal(t, set.find("what"), (*subQueue)(nil))

	for _, p := range pairs {
		k, v := set.next()
		assert.Equal(t, k, p.key)
		assert.Equal(t, v, p.value)
	}

	k, v = set.next()
	assert.Equal(t, k, "")
	assert.Equal(t, v, (*subQueue)(nil))

	const someKey = "something"
	set.add(someKey, nil)
	assertx.Panic(t, func() {
		set.add(someKey, nil)
	}, "key already in set: "+someKey)
}
