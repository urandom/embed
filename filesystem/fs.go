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

type FileSystem struct {
	// Fallback instructs the file system to fall back to the operating system
	// if a file hasn't beed aded to it.
	Fallback bool

	sync.RWMutex
	root node
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

func New() *FileSystem {
	return &FileSystem{
		root: node{"", map[string]node{}, dirStat(""), ""},
	}
}

func (fs *FileSystem) Add(name string, size int64, mode os.FileMode, modTime time.Time, data string) error {
	fs.Lock()
	defer fs.Unlock()

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

func (fs *FileSystem) Open(name string) (http.File, error) {
	fs.RLock()
	defer fs.RUnlock()

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
