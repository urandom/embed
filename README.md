# embed [![Build Status](https://travis-ci.org/urandom/embed.png?branch=master)](https://travis-ci.org/urandom/embed) [![GoDoc](http://godoc.org/github.com/urandom/embed?status.png)](http://godoc.org/github.com/urandom/embed)

Embedding files into Go programs

## Synopsis

Embed provides an [http.FileSystem](https://godoc.org/net/http#FileSystem) implementation. The [FileSystem](https://godoc.org/github.com/urandom/embed/filesystem#FileSystem) allows arbitrary file data to be added to it, and optionally fall back to the operating system when a requested named file isn't found.

An embed command is provided for easy insertion of data into the FileSystem. It generates a Go file that includes the contents of all files or directories passed to it. It also supports directory recursion view  the '/...' suffix.
