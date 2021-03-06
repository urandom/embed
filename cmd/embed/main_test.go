package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"strings"
	"testing"
)

type call struct {
	name string
	size string
	mode string
	data string
}

func TestWrite(t *testing.T) {
	buf := &buffer{}

	cases := []struct {
		header header
		files  []string
		calls  []call
	}{
		{
			header{"test", "Test", "", false},
			[]string{"testdata/..."},
			[]call{
				{"\"testdata/1\"", "11", "420", "\"1234567890\\n\""},
				{"\"testdata/2\"", "11", "308", "\"0987654321\\n\""},
				{"\"testdata/foo.go\"", "65", "260", "\"package main\\n\\nimport \\\"fmt\\\"\\n\\nfunc main() {\\n\\tfmt.Println(\\\"test\\\")\\n}\\n\""},
				{"\"testdata/vmlinuz\"", "20", "267", "\"MZ\\xea\\a\\x00\\xc0\\a\\x8cȎ؎\\xc0\\x8e\\xd01\\xe4\\xfb\\xfc\\xbe\""},
			},
		},
		{
			header{"test2", "Test2", "some,tag", true},
			[]string{"testdata/1", "testdata/vmlinuz"},
			[]call{
				{"\"testdata/1\"", "11", "420", "\"1234567890\\n\""},
				{"\"testdata/vmlinuz\"", "20", "267", "\"MZ\\xea\\a\\x00\\xc0\\a\\x8cȎ؎\\xc0\\x8e\\xd01\\xe4\\xfb\\xfc\\xbe\""},
			},
		},
		{
			header{"test", "Test", "", false},
			[]string{"testdata"},
			[]call{
				{"\"testdata/1\"", "11", "420", "\"1234567890\\n\""},
				{"\"testdata/2\"", "11", "308", "\"0987654321\\n\""},
				{"\"testdata/foo.go\"", "65", "260", "\"package main\\n\\nimport \\\"fmt\\\"\\n\\nfunc main() {\\n\\tfmt.Println(\\\"test\\\")\\n}\\n\""},
				{"\"testdata/vmlinuz\"", "20", "267", "\"MZ\\xea\\a\\x00\\xc0\\a\\x8cȎ؎\\xc0\\x8e\\xd01\\xe4\\xfb\\xfc\\xbe\""},
			},
		},
		{
			header{"test2", "Test2", "", true},
			[]string{},
			[]call{},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			buf.Reset()

			writeData(buf, tc.header, tc.files, false, false)

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

			if tc.header.Tags == "" {
				if strings.Contains(buf.String(), "// +build") {
					t.Fatalf("A build tag wasn't expected")
				}
			} else {
				if !strings.Contains(buf.String(), "// +build "+tc.header.Tags) {
					t.Fatalf("A build tag was expected")
				}
			}

			if f.Name.Name != tc.header.Pkg {
				t.Fatalf("expected package name %s, got %s", tc.header.Pkg, f.Name.Name)
			}

			if funcDeck, ok := f.Decls[1].(*ast.FuncDecl); ok {
				if funcDeck.Name.Name != tc.header.Function {
					t.Fatalf("expected function name %s, got %s", tc.header.Function, funcDeck.Name.Name)
				}
			} else {
				t.Fatalf("Expected a func declaration")
			}

			addCalls := 0
			ast.Inspect(f, func(n ast.Node) bool {
				if callExpr, ok := n.(*ast.CallExpr); ok {
					selX, ok := callExpr.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					ident, ok := selX.X.(*ast.Ident)
					if !ok || ident.Name != "fs" || selX.Sel.Name != "Add" {
						return true
					}

					call := tc.calls[addCalls]
					addCalls++

					if len(callExpr.Args) != 5 {
						t.Fatalf("expected 5 arguments, got %d", len(callExpr.Args))
					}

					first, ok := callExpr.Args[0].(*ast.BasicLit)
					if !ok {
						t.Fatalf("Expected a basic literal")
					}

					if first.Kind != token.STRING {
						t.Fatalf("Expected a string")
					}

					if first.Value != call.name {
						t.Fatalf("Expected %s, got %s", call.name, first.Value)
					}

					second, ok := callExpr.Args[1].(*ast.BasicLit)
					if !ok {
						t.Fatalf("Expected a basic literal")
					}

					if second.Kind != token.INT {
						t.Fatalf("Expected an int ")
					}

					if second.Value != call.size {
						t.Fatalf("Expected %s, got %s", call.size, second.Value)
					}

					third, ok := callExpr.Args[2].(*ast.CallExpr)
					if !ok {
						t.Fatalf("Expected a call expression")
					}

					if len(third.Args) != 1 {
						t.Fatalf("Expected 1 argument, got %d", len(third.Args))
					}

					lit, ok := third.Args[0].(*ast.BasicLit)
					if !ok {
						t.Fatalf("Expected a basic literal")
					}

					if lit.Value != call.mode {
						t.Fatalf("Expected %s for %s, got %s", call.name, call.mode, lit.Value)
					}

					fourth, ok := callExpr.Args[3].(*ast.CallExpr)
					if !ok {
						t.Fatalf("Expected a call expression")
					}

					if len(fourth.Args) != 2 {
						t.Fatalf("Expected 2 argument, got %d", len(fourth.Args))
					}

					selX, ok = fourth.Fun.(*ast.SelectorExpr)
					if !ok {
						t.Fatalf("Expected a selector expression")
					}

					if ident, ok := selX.X.(*ast.Ident); ok {
						if ident.Name != "time" {
							t.Fatalf("Expected 'time', got %s", ident.Name)
						}
					} else {
						t.Fatalf("Expected an identifier")
					}

					if selX.Sel.Name != "Unix" {
						t.Fatalf("Expected 'Unix', got %s", selX.Sel.Name)
					}

					fifth, ok := callExpr.Args[4].(*ast.BasicLit)
					if !ok {
						t.Fatalf("Expected a basic literal")
					}

					if fifth.Kind != token.STRING {
						t.Fatalf("Expected a string")
					}

					if fifth.Value != call.data {
						t.Fatalf("Expected %s, got %s", call.data, fifth.Value)
					}
				}

				return true
			})

			if addCalls != len(tc.calls) {
				t.Fatalf("expected %d fs.Add calls, got %d", len(tc.calls), addCalls)
			}
		})
	}

}

type buffer struct {
	bytes.Buffer
}

func (cb *buffer) Close() error {
	return nil
}

func init() {
	if err := os.Chmod("testdata/1", 0644); err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod("testdata/2", 0464); err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod("testdata/foo.go", 772); err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod("testdata/vmlinuz", 267); err != nil {
		log.Fatal(err)
	}
}
