// Code generated by "esc -o=esc/esc.go -pkg=esc -ignore=esc -include=\.sql$ -private -modtime=0 ."; DO NOT EDIT.

package esc

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type _escLocalFS struct{}

var _escLocal _escLocalFS

type _escStaticFS struct{}

var _escStatic _escStaticFS

type _escDirectory struct {
	fs   http.FileSystem
	name string
}

type _escFile struct {
	compressed string
	size       int64
	modtime    int64
	local      string
	isDir      bool

	once sync.Once
	data []byte
	name string
}

func (_escLocalFS) Open(name string) (http.File, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_escStaticFS) prepare(name string) (*_escFile, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		f.name = path.Base(name)
		if f.size == 0 {
			return
		}
		var gr *gzip.Reader
		b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(f.compressed))
		gr, err = gzip.NewReader(b64)
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (fs _escStaticFS) Open(name string) (http.File, error) {
	f, err := fs.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (dir _escDirectory) Open(name string) (http.File, error) {
	return dir.fs.Open(dir.name + name)
}

func (f *_escFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*_escFile
	}
	return &httpFile{
		Reader:   bytes.NewReader(f.data),
		_escFile: f,
	}, nil
}

func (f *_escFile) Close() error {
	return nil
}

func (f *_escFile) Readdir(count int) ([]os.FileInfo, error) {
	if !f.isDir {
		return nil, fmt.Errorf(" escFile.Readdir: '%s' is not directory", f.name)
	}

	fis, ok := _escDirs[f.local]
	if !ok {
		return nil, fmt.Errorf(" escFile.Readdir: '%s' is directory, but we have no info about content of this dir, local=%s", f.name, f.local)
	}
	limit := count
	if count <= 0 || limit > len(fis) {
		limit = len(fis)
	}

	if len(fis) == 0 && count > 0 {
		return nil, io.EOF
	}

	return fis[0:limit], nil
}

func (f *_escFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *_escFile) Name() string {
	return f.name
}

func (f *_escFile) Size() int64 {
	return f.size
}

func (f *_escFile) Mode() os.FileMode {
	return 0
}

func (f *_escFile) ModTime() time.Time {
	return time.Unix(f.modtime, 0)
}

func (f *_escFile) IsDir() bool {
	return f.isDir
}

func (f *_escFile) Sys() interface{} {
	return f
}

// _escFS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func _escFS(useLocal bool) http.FileSystem {
	if useLocal {
		return _escLocal
	}
	return _escStatic
}

// _escDir returns a http.Filesystem for the embedded assets on a given prefix dir.
// If useLocal is true, the filesystem's contents are instead used.
func _escDir(useLocal bool, name string) http.FileSystem {
	if useLocal {
		return _escDirectory{fs: _escLocal, name: name}
	}
	return _escDirectory{fs: _escStatic, name: name}
}

// _escFSByte returns the named file from the embedded assets. If useLocal is
// true, the filesystem's contents are instead used.
func _escFSByte(useLocal bool, name string) ([]byte, error) {
	if useLocal {
		f, err := _escLocal.Open(name)
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadAll(f)
		_ = f.Close()
		return b, err
	}
	f, err := _escStatic.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.data, nil
}

// _escFSMustByte is the same as _escFSByte, but panics if name is not present.
func _escFSMustByte(useLocal bool, name string) []byte {
	b, err := _escFSByte(useLocal, name)
	if err != nil {
		panic(err)
	}
	return b
}

// _escFSString is the string version of _escFSByte.
func _escFSString(useLocal bool, name string) (string, error) {
	b, err := _escFSByte(useLocal, name)
	return string(b), err
}

// _escFSMustString is the string version of _escFSMustByte.
func _escFSMustString(useLocal bool, name string) string {
	return string(_escFSMustByte(useLocal, name))
}

var _escData = map[string]*_escFile{

	"/20190504194418_initial_schema.down.sql": {
		name:    "20190504194418_initial_schema.down.sql",
		local:   "20190504194418_initial_schema.down.sql",
		size:    283,
		modtime: 0,
		compressed: `
H4sIAAAAAAAC/1zPwW7DIAwG4DtP4QfYG+SUpmxCWtdqyWE7IQqegupABiZ7/R1Cqig3/9/B9n+Sb+qj
EeL8eb3B0J7eJahXkF+qH3oBAGBHEwJSfllTyRwnbeM0meAq/pbIWGdTOCacyW+wmOTNnbbIf57tqDk+
MFS6U7QPdLpkTNuZdb8mn/lAPvzESglnNIzu8E62I7pCOxdd23ftWT57ft92NcFYizlrwgWpEaK7Xi5q
aMR/AAAA//955cCYGwEAAA==
`,
	},

	"/20190504194418_initial_schema.up.sql": {
		name:    "20190504194418_initial_schema.up.sql",
		local:   "20190504194418_initial_schema.up.sql",
		size:    7371,
		modtime: 0,
		compressed: `
H4sIAAAAAAAC/+xYW2/bthd/96c4/yc3+CdF8lx0gJOordFE6WwHXTAMBCUe2Vx4UUkqsTfsuw/UxZEs
yk4LFFuR5SnyufGc37mRJyewWHELGRcI3IItkt8xdeA0pCuqlgiFclyAWyFk3FgHBqkAhrnQG4nKgc5K
YqLd69HJCVxqUNpziQ1o5UkWwaYrlBQkXxrquFYWDErKFVdLSLWy3DqvqjHFrdck0Vq6RFhRCwmi8jL6
Adnr0eg8ej+N34xGo4tZNFlEsLj7FAFNU7SWCHxAAZM5RPHtNbwaAQCM8QHNRiscH1fftkhsaniCpvlF
aoaGOr39ITGaspRa98RDmeRqPDpqW56cX0VlqBQKW5vjDBK+5MrB+yiOZpNFdAnnd3AZvZvcXi382aaX
UbyYLu7g02x6PZndwcforjKSGqQOGaEOHJdoHZW5+2MrG998fnUE8c0C4turq0qkyNnXiFQyFg15OmdD
hdt4+vNtVOlVVCI4XLsde4l2ZIhGU8cffDZogVTtEHODGV8HVRZCoCsp9QE9Hl1IdzzgS6UNslLm19+2
3o7//Gu8ozwtrNOS6EeFxj6fX2r2FdwGl4WgB/VXIloLph8VcNW4K6h1JJOh2OTUWCQbXbgiGYosrp2h
RPAMPahtRCu6oVkmkKCiiUA2oMQ6pNLLBw5RmISqA/LuEWsMdx02WogaxQFIt1zt2IToDDNaCLdDrs6/
0oVgpK7koVAxbnNBN+SRGt9/7FBES1cPcfl604UjrKgaW+DUtaKMC+fTr6+n5KrIRHB1P2QrRyO584Ve
cR1OTd/liKSbUuCA5ZTmQ4ZbHERyRdKVz/O+p22+HE2Kyvn+vZ+xVOhtBwCt+VBqhwcOV/EQSdfD9moe
y9VS4IFw2I1MtDhgtGZ6lrMNr/e3UT7sssT9liVdE4Fq6VZ7lCR+KDGSrwy1h+LX5SU5dQ6NelY3u/gQ
XXyEV3Vz/99bGI+PjtuUgbw4jxafoyiGU6CKwdnpaVAqEOMDgr2a/OktnB6VU7se2tP4MvplO7RJPQsJ
Z2u/sjwN85oQmPdVz0+1lFSxH2js1761Jv8sehfNovgimrcc56wn2WxjnfYeCGo3MuTJYBnem7gfuyeW
fqC/FGXpv4D4qkIGmkfpf3Cklg5pE5rVyHiAUpKq9a4d82NvOVAdVeTb+KlCNhg2sOyq6eFHC6cN5oK/
ZBCd4cslBqHShi+bZltvv+WmgzbXygY37FQXKrj+fO+MaGEZKOsO0p2SHlTSSqiOtI/iZH7ROkCVTA/U
cL9LvYxUGrhgPVBR4PORpBIDUG4j2alvKrHBoxXqnrpekbtH7tIVcfoe1Q+ETX3sA5fg7mW3odR33fIq
U/odgqokELfJg0AazAza1bA4rnNuNh1/huduBwOyda3BcweiLb0PZiJ0eo+sXIr+ATCfhcyo1xrqbYII
bt2LaA/cody/nYeeqeoocZXpF91EO48Qe14kqlm7k4TH3zZtt688hUXWdr7xsrMXDzi7uzt7n9vdql0H
Ayq6ldJWsH+K1Hesm3i+mE2m8QLS1T0pFP9SlG9O9eWrZPJ/T/+VXxeTeQSfP0RxwM/pvCruxYfyUhdd
zSM4A1TsqKPj/x0tO44+Q8cRvIWz8it0GWyXxtBU3Kmfw5PRYF5VyAu5KbZT0AdpfwrWYWyr6Obi/udG
hoJuAut2fVMljGdZY71x9SxUkANV/i1l3k+sXgoEdudAmgxv0AGF3Yjv09rl7CesTVfICvFfxn6XjE2N
VgTXuUFruQ4uff+u5O2nQyB7QzkznL4hleH8DeoNJfDN9fV08Wb0dwAAAP//Am8mOcscAAA=
`,
	},

	"/": {
		name:  "/",
		local: `.`,
		isDir: true,
	},
}

var _escDirs = map[string][]os.FileInfo{

	".": {
		_escData["/20190504194418_initial_schema.down.sql"],
		_escData["/20190504194418_initial_schema.up.sql"],
	},
}
