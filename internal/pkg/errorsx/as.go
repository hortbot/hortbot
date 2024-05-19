package errorsx

import "errors"

// As is a wrapper around errors.As with a type parameter.
func As[T error](err error) (T, bool) {
	var target T
	if ok := errors.As(err, &target); ok {
		return target, true
	}
	var zero T
	return zero, false
}
