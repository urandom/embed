package filesystem

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
)

type file struct {
	*strings.Reader
	stat os.FileInfo
}

type dir struct {
	pos   int
	stat  os.FileInfo
	files []os.FileInfo
}

func newFile(data string, stat os.FileInfo) http.File {
	return file{strings.NewReader(data), stat}
}

func newDir(stat os.FileInfo, files []os.FileInfo) http.File {
	return &dir{0, stat, files}
}

func (f file) Close() error {
	return nil
}

func (f file) Stat() (os.FileInfo, error) {
	return f.stat, nil
}

func (f file) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.Wrap(os.ErrInvalid, "not a directory")
}

func (d *dir) Close() error {
	d.pos = 0
	return nil
}

func (d *dir) Seek(int64, int) (int64, error) {
	return 0, errors.Wrap(os.ErrInvalid, "seek "+d.stat.Name()+"  is a directory")
}

func (d dir) Stat() (os.FileInfo, error) {
	return d.stat, nil
}

func (d dir) Read(b []byte) (int, error) {
	return 0, errors.Wrap(os.ErrInvalid, "read "+d.stat.Name()+" is a directory")
}

func (d *dir) Readdir(count int) ([]os.FileInfo, error) {
	stats := []os.FileInfo{}

	if count <= 0 {
		return d.files, nil
	}

	count += d.pos
	if count > len(d.files) {
		count = len(d.files)
	}

	for ; d.pos < count; d.pos++ {
		stats = append(stats, d.files[d.pos])
	}

	if len(stats) == 0 {
		return stats, io.EOF
	}

	return stats, nil
}
