package must

func Must[T any](val T, err error) T {
	NilError(err)
	return val
}

func NilError(err error) {
	if err != nil {
		panic(err)
	}
}
