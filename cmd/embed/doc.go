/*
Embed is a tool for creating filesystem entries by inserting file data as
safely-quoted strings in the generated Go file.

The desired files or directories are passed as arguments to the command:

	embed some_file ./a/directory recursive/insertion/...

By default, it will generated a file called file_data.go:

	// DO NOT EDIT ** This file was generated with github.com/urandom/embed ** DO NOT EDIT //

	package main

	import (
		"fmt"
		"os"
		"time"

		"github.com/pkg/errors"
		"github.com/urandom/embed/filesystem"
	)

	func NewFileSystem() (*filesystem.FileSystem, error) {
		fs := filesystem.New()

		if err := fs.Add("some_file", SIZE, os.FileMode(MODE), time.Unix(TIMESTAMP, 0), "DATA"); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("packing file some_file"))
		}

		...

		return fs, nil
	}

The output file, function and package names, as well as build tags can be set
via flags.

*/
package main
