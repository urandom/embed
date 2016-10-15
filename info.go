package embed

import (
	"os"
	"time"
)

type info struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi info) Name() string {
	return fi.name
}

func (fi info) Size() int64 {
	return fi.size
}

func (fi info) Mode() os.FileMode {
	return fi.mode
}

func (fi info) ModTime() time.Time {
	return fi.modTime
}

func (fi info) IsDir() bool {
	return fi.mode.IsDir()
}

func (fi info) Sys() interface{} {
	return nil
}
