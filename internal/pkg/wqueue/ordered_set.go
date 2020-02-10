package wqueue

// Note: not safe for concurrent use.
type orderedSet struct {
	items   []pair
	mapping map[string]*subQueue
}

func newOrderedSet() *orderedSet {
	return &orderedSet{
		mapping: make(map[string]*subQueue),
	}
}

type pair struct {
	key   string
	value *subQueue
}

func (o *orderedSet) add(key string, value *subQueue) {
	if _, ok := o.mapping[key]; ok {
		panic("key already in set: " + key) // Not technically a set, since only one of each key is allowed.
	}
	o.mapping[key] = value
	o.items = append(o.items, pair{key: key, value: value})
}

func (o *orderedSet) find(key string) *subQueue {
	return o.mapping[key]
}

func (o *orderedSet) next() (key string, value *subQueue) {
	if len(o.items) == 0 {
		return "", nil
	}

	p := o.items[0]
	o.items[0] = pair{} // Prevent leaks.
	o.items = o.items[1:]
	delete(o.mapping, p.key)
	return p.key, p.value
}
