// Package assertx contains extensions to the gotest.tools/assert package.
package assertx

import "gotest.tools/v3/assert"

type helperT interface {
	Helper()
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . testingT

type testingT interface { //nolint
	assert.TestingT
	helperT
}

// Panic checks that a function panics with the given value. An untyped nil
// panic value represents no panic.
func Panic(t assert.TestingT, fn func(), want interface{}, msgAndArgs ...interface{}) {
	if ht, ok := t.(helperT); ok {
		ht.Helper()
	}

	var got interface{}

	func() {
		defer func() {
			got = recover()
		}()
		fn()
	}()

	assert.Equal(t, got, want, msgAndArgs...)
}
