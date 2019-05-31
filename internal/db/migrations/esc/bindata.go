package esc

// AssetNames provides a go-bindata like interface to use with esc until
// golang-migrate supports http.FileSystem.
func AssetNames() []string {
	names := make([]string, 0, len(_escData))
	for name, entry := range _escData {
		if !entry.isDir {
			names = append(names, name[1:])
		}
	}
	return names
}

// Asset provides a go-bindata like interface to use with esc until
// golang-migrate supports http.FileSystem.
func Asset(name string) ([]byte, error) {
	return _escFSByte(false, "/"+name)
}
