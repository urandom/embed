/*
Package embed contains an implementation of http.FileSystem that stores the
file data in itself. It also includes a tool for generating Go code that adds
files from the operating system to the http.FileSystem implementation.

The following subpackages contain:

	* filesystem - contains the implementation of http.FileSystem.
	* cmd/embed - contains the tool that generates the filesystem inserts.

This package itself is used for documentation.
*/
package embed
