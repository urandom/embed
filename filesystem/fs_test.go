package filesystem

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
)

var (
	now   = time.Now()
	files = []struct {
		name   string
		stat   os.FileInfo
		data   string
		query  string
		exists bool
		inOS   bool
	}{
		{"foo", info{name: "foo", size: 4, mode: 0x1a4, modTime: now}, "1234", "foo", true, false},
		{"bar", info{name: "bar", size: 8, mode: 0x1a5, modTime: now}, "98765432", "/bar", true, false},
		{"d/alpha", info{name: "alpha", size: 8, mode: 0x1a5, modTime: now}, "98765432", "d/alpha", true, false},
		{"d/beta", info{name: "beta", size: 8, mode: 0x1a5, modTime: now}, "98765432", "./d/beta", true, false},
		{"d/gamma", info{name: "gamma", size: 8, mode: 0x1a5, modTime: now}, "98765432", "/d/gamma", true, false},
		{"", info{}, "", "fs_test.go", false, true},
		{"", info{}, "", "fs_test.stop", false, false},
	}
)

func TestFiles(t *testing.T) {
	fs := New()
	fallback := New()
	fallback.Fallback = true

	for i, fs := range []*FileSystem{fs, fallback} {
		for j, tc := range files {
			t.Run(fmt.Sprintf("case %d-%d", i, j), func(t *testing.T) {
				if tc.name != "" {
					err := fs.Add(tc.name, tc.stat.Size(), tc.stat.Mode(), tc.stat.ModTime(), tc.data)
					if err != nil {
						t.Fatalf("didn't expect error %+v", err)
					}

					err = fs.Add(tc.name, tc.stat.Size(), tc.stat.Mode(), tc.stat.ModTime(), tc.data)
					if err == nil {
						t.Fatalf("file already exist, should've gotten an error")
					} else if !os.IsExist(errors.Cause(err)) {
						t.Fatalf("expected %v, got %+v", os.ErrExist, err)
					}
				}

				f, err := fs.Open(tc.query)
				if tc.exists || fs.Fallback && tc.inOS {
					if err != nil {
						t.Fatalf("opening file: %+v", err)
					}
				} else {
					if !os.IsNotExist(errors.Cause(err)) {
						t.Fatalf("expected ErrNotExist, got %+v", err)
					}

					return
				}

				b := make([]byte, tc.stat.Size())

				_, err = f.Read(b)
				if err != nil {
					t.Fatalf("reading file: %+v", err)
				}

				if string(b) != tc.data {
					t.Fatalf("expected data %s, got %s", tc.data, string(b))
				}

				stat, err := f.Stat()
				if err != nil {
					t.Fatalf("file stat: %+v", err)
				}

				if !fs.Fallback || !tc.inOS {
					if stat.Name() != tc.stat.Name() {
						t.Fatalf("expected name %s, got %s", tc.stat.Name(), stat.Name())
					}

					if stat.Size() != tc.stat.Size() {
						t.Fatalf("expected size %s, got %s", tc.stat.Size(), stat.Size())
					}

					if stat.Mode() != tc.stat.Mode() {
						t.Fatalf("expected mode %s, got %s", tc.stat.Mode(), stat.Mode())
					}

					if stat.ModTime() != tc.stat.ModTime() {
						t.Fatalf("expected mod time %s, got %s", tc.stat.ModTime(), stat.ModTime())
					}
				}
			})
		}
	}
}

func TestDir(t *testing.T) {
	fs := New()

	for _, f := range files {
		if f.name != "" {
			fs.Add(f.name, f.stat.Size(), f.stat.Mode(), f.stat.ModTime(), f.data)
		}
	}

	cases := []struct {
		name   string
		exists bool
		n      int
		again  bool
		names  []string
		n2     int
		names2 []string
		eof    bool
	}{
		{"", true, -1, false, []string{"bar", "d", "foo"}, 0, nil, false},
		{"", true, 0, false, []string{"bar", "d", "foo"}, 0, nil, false},
		{"", true, 1, true, []string{"bar"}, 1, []string{"d"}, false},
		{"", true, 2, true, []string{"bar", "d"}, 1, []string{"foo"}, false},
		{"", true, 1, true, []string{"bar"}, 2, []string{"d", "foo"}, false},

		{".", true, 0, false, []string{"bar", "d", "foo"}, 0, nil, false},
		{"/", true, 0, false, []string{"bar", "d", "foo"}, 0, nil, false},
		{"/d", true, 0, false, []string{"alpha", "beta", "gamma"}, 0, nil, false},
		{"./d", true, 0, false, []string{"alpha", "beta", "gamma"}, 0, nil, false},
		{"d", true, 0, false, []string{"alpha", "beta", "gamma"}, 0, nil, false},
		{"d/", true, 0, false, []string{"alpha", "beta", "gamma"}, 0, nil, false},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			f, err := fs.Open(tc.name)

			if err != nil {
				if os.IsNotExist(errors.Cause(err)) != tc.exists {
					t.Fatalf("expected to exist: %v, got %+v", tc.exists, err)
				}
			}

			stats, err := f.Readdir(tc.n)
			if err != nil {
				t.Fatalf("err: %+v", err)
			}

			if len(tc.names) != len(stats) {
				t.Fatalf("expected %d entries, got %d", len(tc.names), len(stats))
			}

			for i, n := range tc.names {
				if stats[i].Name() != n {
					t.Fatalf("expected %s, got %s", n, stats[i].Name())
				}
			}

			if tc.again {
				stats, err = f.Readdir(tc.n2)
				if err != nil {
					if errors.Cause(err) != io.EOF || !tc.eof {
						t.Fatalf("err: %+v", err)
					}
				}

				for i, n := range tc.names2 {
					if stats[i].Name() != n {
						t.Fatalf("expected %s, got %s", n, stats[i].Name())
					}
				}
			}
		})
	}
}
