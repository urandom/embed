package filesystem

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// A FileSystem is used to store and access file data. It implements the
// http.FileSystem interface to be used effortlessly with a FileServer handler.
//
// Since all file data can be provided as a string literal in the Go source,
// such entries can be effectively embedded into the resultant binary.
type FileSystem struct {
	// Fallback instructs the filesystem to fall back to the operating system
	// if a file hasn't beed aded to it.
	Fallback bool

	mutex *sync.RWMutex
	root  node
}

type node struct {
	name     string
	children map[string]node
	stat     os.FileInfo
	data     string
}

type payload struct {
	stat os.FileInfo
	data string
}

// New creates a fresh instance of a FileSystem
func New() *FileSystem {
	return &FileSystem{
		mutex: &sync.RWMutex{},
		root:  node{"", map[string]node{}, dirStat(""), ""},
	}
}

// Add inserts a new named file representation into the filesystem. The size,
// mode and modTime parameters represent it's byte length, mode bits, and
// modification time, respectively. The file data is represented as a string,
// which is usually created using the fmt package's '%q' verb.
//
// If the file name represents a relative path, it will be converted to an
// absolute path where it's base directory is root of the filesystem. E.g.:
//
//	relative/path		->	/relative/path
//	./relative/path		->	/relative/path
//	/absolute/path		->	/absolute/path
func (fs *FileSystem) Add(
	name string, size int64, mode os.FileMode, modTime time.Time, data string,
) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	name = path.Clean(filepath.ToSlash(name))
	if path.IsAbs(name) {
		name = name[1:]
	}

	parts := strings.Split(name, "/")
	stat := info{parts[len(parts)-1], size, mode, modTime}
	n := &fs.root
	for i, p := range parts {
		if i == len(parts)-1 {
			// Leaf
			if _, ok := n.children[p]; ok {
				return errors.Wrap(os.ErrExist, "non-dir node already exists")
			}
			var children map[string]node
			if stat.IsDir() {
				children = map[string]node{}
			}

			n.children[p] = node{p, children, stat, data}
			break
		}

		if c, ok := n.children[p]; ok {
			if !c.stat.IsDir() {
				return errors.Wrap(os.ErrExist, "non-dir node already exists")
			}
			n = &c
		} else {
			c := node{p, map[string]node{}, dirStat(p), ""}
			n.children[p] = c
			n = &c
		}
	}

	return nil
}

// Opens a previously inserted named file. If such a file hasn't been added, it
// may optionally fall back to accessing the file with the same path in the
// operating system.
func (fs *FileSystem) Open(name string) (http.File, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	name = path.Clean(filepath.ToSlash(name))
	if path.IsAbs(name) {
		name = name[1:]
		if name == "" {
			name = "."
		}
	}

	parts := strings.Split(name, "/")
	n := fs.root
	for _, p := range parts {
		if p == "." {
			continue
		} else {
			c, ok := n.children[p]
			if !ok {
				if fs.Fallback {
					f, err := os.Open(name)
					if err != nil {
						err = errors.Wrap(err, "falling back to OS")
					}

					return f, err
				}
				return nil, errors.Wrap(os.ErrNotExist, "opening file")
			}

			n = c
		}
	}

	if n.stat.IsDir() {
		files := []os.FileInfo{}
		names := make([]string, 0, len(n.children))

		for name := range n.children {
			names = append(names, name)
		}

		sort.Strings(names)

		for _, name := range names {
			files = append(files, n.children[name].stat)
		}

		return newDir(n.stat, files), nil
	} else {
		return newFile(n.data, n.stat), nil
	}
}

func dirStat(name string) os.FileInfo {
	return info{name, 4096, 0x800001ed, time.Now()}
}
