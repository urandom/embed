package main

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"testing"
)

func TestWrite(t *testing.T) {
	buf := &buffer{}

	writeData(buf, header{"test", "Test", "", false}, []string{"testdata/..."}, false, false)

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "file_data.go", buf.Bytes(), 0)
	if err != nil {
		t.Fatalf("parsing expr: %+v", err)
	}

	conf := types.Config{Importer: importer.Default()}
	_, err = conf.Check("hello", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatalf("checking: %+v", err)
	}

	totalFiles, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("reading testdata dir: %+v", err)
	}

	addCalls := 0
	ast.Inspect(f, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selX, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selX.X.(*ast.Ident)
		if !ok || ident.Name != "fs" || selX.Sel.Name != "Add" {
			return true
		}

		addCalls++

		return true
	})

	if addCalls != len(totalFiles) {
		t.Fatalf("expected %d fs.Add calls, got %d", len(totalFiles), addCalls)
	}
}

type buffer struct {
	bytes.Buffer
}

func (cb *buffer) Close() error {
	return nil
}
