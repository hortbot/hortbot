package version

var version string

// Version returns a compile time version string, or "(devel)" if unset.
func Version() string {
	if version == "" {
		return "(devel)"
	}
	return version
}
