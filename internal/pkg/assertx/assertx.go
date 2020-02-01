// Package assertx contains extensions to the gotest.tools/assert package.
package assertx

import "gotest.tools/v3/assert"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . TestingT

type TestingT interface {
	assert.TestingT
	Helper()
}

// Panic checks that a function panics with the given value. An untyped nil
// panic value represents no panic.
func Panic(t TestingT, fn func(), want interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	var got interface{}

	func() {
		defer func() {
			got = recover()
		}()
		fn()
	}()

	assert.Equal(t, got, want, msgAndArgs...)
}
