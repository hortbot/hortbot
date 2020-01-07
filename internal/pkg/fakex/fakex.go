// Package fakex provides extensions for counterfeiter fakes.
package fakex

import (
	"reflect"
	"strings"
	"testing"
)

// StubNotImplemented stubs out unset stubs in the provided fake with functions
// that will fail the test if called. This is useful to ensure that no function
// other than the expected ones are called, without manually updating things when
// the interface changes.
//
// Do not use this function if the fake's "Returns" feature is used or the default
// zero return is wanted, as the stub will take precedence.
//
// If passed bad arguments, this function may panic.
func StubNotImplemented(t testing.TB, fake interface{}) {
	fakeValue := reflect.ValueOf(fake)
	fakeName := fakeValue.Type().String()

	fakeValue = fakeValue.Elem()
	fakeType := fakeValue.Type()

	for i := 0; i < fakeValue.NumField(); i++ {
		f := fakeType.Field(i)
		if f.Type.Kind() != reflect.Func || !strings.HasSuffix(f.Name, "Stub") {
			continue
		}

		fv := fakeValue.Field(i)
		if !fv.IsZero() {
			continue
		}

		out := make([]reflect.Value, f.Type.NumOut())

		for j := range out {
			out[j] = reflect.Zero(f.Type.Out(j))
		}

		fn := reflect.MakeFunc(f.Type, func(args []reflect.Value) (results []reflect.Value) {
			t.Helper()
			t.Fatalf("(%s).%s not implemented", fakeName, f.Name)
			return out
		})

		fv.Set(fn)
	}
}
