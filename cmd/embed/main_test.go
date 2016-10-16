package main

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestWrite(t *testing.T) {
	buf := &buffer{}

	writeData(buf, header{"test", "Test", "", false}, []string{"testdata/..."}, false, false)

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "file_data.go", buf.Bytes(), 0)
	if err != nil {
		t.Fatalf("parsing expr: %+v\n", err)
	}

	conf := types.Config{Importer: importer.Default()}
	_, err = conf.Check("hello", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatalf("checking: %+v\n", err)
	}
}

type buffer struct {
	bytes.Buffer
}

func (cb *buffer) Close() error {
	return nil
}
