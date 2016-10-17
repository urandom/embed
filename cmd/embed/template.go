package main

import "text/template"

type header struct {
	Pkg      string
	Function string
	Tags     string
	Fallback bool
}

type file struct {
	Name    string
	Data    string
	Size    int64
	Mode    uint32
	ModTime int64
}

var (
	headerTmpl      = template.Must(template.New("gen-header").Parse(headerData))
	emptyHeaderTmpl = template.Must(template.New("gen-empty-header").Parse(emptyHeaderData))
	fileTmpl        = template.Must(template.New("gen-file").Parse(fileData))
	footerTmpl      = template.Must(template.New("gen-footer").Parse(footerData))
)

const (
	headerData = `
{{- if .Tags }}// +build {{ .Tags }}
{{- end }}

// DO NOT EDIT ** This file was generated with github.com/urandom/embed ** DO NOT EDIT //

package {{ .Pkg }}

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/urandom/embed/filesystem"
)

func {{ .Function }}() (*filesystem.FileSystem, error) {
	fs := filesystem.New()
{{ if .Fallback }}
	fs.Fallback = true
{{ end -}}
`

	emptyHeaderData = `
{{- if .Tags }}// +build {{ .Tags }}
{{- end }}

// DO NOT EDIT ** This file was generated with github.com/urandom/embed ** DO NOT EDIT //

package {{ .Pkg }}

import (
	"github.com/urandom/embed/filesystem"
)

func {{ .Function }}() (*filesystem.FileSystem, error) {
	fs := filesystem.New()
{{ if .Fallback }}
	fs.Fallback = true
{{ end -}}
`

	fileData = `
	if err := fs.Add("{{ .Name }}", {{ .Size }}, os.FileMode({{ .Mode }}), time.Unix({{ .ModTime }}, 0), {{ .Data }}); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("packing file {{ .Name }}"))
	}
`

	footerData = `
	return fs, nil
}
`
)
