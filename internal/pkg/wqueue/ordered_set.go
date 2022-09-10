package wqueue

import "fmt"

// Note: not safe for concurrent use.
type orderedSet[K comparable, V any] struct {
	items   []pair[K, V]
	mapping map[K]V
}

func newOrderedSet[K comparable, V any]() *orderedSet[K, V] {
	return &orderedSet[K, V]{
		mapping: make(map[K]V),
	}
}

type pair[K comparable, V any] struct {
	key   K
	value V
}

func (o *orderedSet[K, V]) add(key K, value V) {
	if _, ok := o.mapping[key]; ok {
		panic(fmt.Sprintf("key already in set: %v", key)) // Not technically a set, since only one of each key is allowed.
	}
	o.mapping[key] = value
	o.items = append(o.items, pair[K, V]{key: key, value: value})
}

func (o *orderedSet[K, V]) find(key K) V {
	return o.mapping[key]
}

func (o *orderedSet[K, V]) next() (key K, value V) {
	if len(o.items) == 0 {
		var zeroK K
		var zeroV V
		return zeroK, zeroV
	}

	p := o.items[0]
	o.items[0] = pair[K, V]{} // Prevent leaks.
	o.items = o.items[1:]
	delete(o.mapping, p.key)
	return p.key, p.value
}
