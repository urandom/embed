package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

var (
	output       string
	input        string
	functionName string
	packageName  string
	buildTags    string
	fatal        bool
	fallback     bool
	verbose      bool
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 && !fallback && input == "" {
		flag.Usage()
		os.Exit(2)
	}

	var out io.WriteCloser
	var err error

	if output == "-" {
		out = os.Stdout
	} else {
		out, err = os.OpenFile(output, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("opening %s for writing: %+v\n", output, err)
		}
	}

	names := flag.Args()
	if input != "" {
		names = processInput(input)
	}

	writeData(out, header{packageName, functionName, buildTags, fallback}, names, fatal, verbose)
}

func processInput(input string) []string {
	names := make([]string, 0, 20)

	var in io.Reader

	if input == "-" {
		in = os.Stdin
	} else {
		f, err := os.Open(input)
		if err != nil {
			log.Fatalf("Error opening input file %s: %+v\n", input, err)
		}

		defer f.Close()

		in = f
	}

	r := bufio.NewReader(in)

	for {
		buf, err := r.ReadSlice('\n')

		end := false
		if err == io.EOF {
			end = true
		} else if err != nil {
			log.Fatalf("Error reading line '%s': %v\n", buf, err)
		}

		index := bytes.IndexByte(buf, '#')
		if index != -1 {
			buf = buf[:index]
		}

		buf = bytes.TrimSpace(buf)

		if len(buf) == 0 {
			if end {
				break
			}
			continue
		}

		names = append(names, string(buf))

		if end {
			break
		}
	}

	return names
}

func writeData(w io.WriteCloser, h header, names []string, fatal, verbose bool) {
	defer func() {
		if err := w.Close(); err != nil {
			log.Fatalf("closing output file: %+v\n", err)
		}
	}()

	buf := bytes.Buffer{}

	defer func() {
		buf.Reset()
		err := footerTmpl.Execute(&buf, nil)
		if err != nil {
			log.Fatalf("executing footer template: %+v\n", err)
		}

		buf.WriteTo(w)
	}()

	errChan := make(chan error)
	go func() {
		for {
			select {
			case err := <-errChan:
				if verbose {
					log.Printf("processing file: %+v\n", err)
				}
			}
		}
	}()

	var headerWritten bool

	for _, name := range names {
		for f := range processFile(name, errChan) {
			if !headerWritten {
				err := headerTmpl.Execute(&buf, h)
				if err != nil {
					log.Fatalf("executing header template: %+v\n", err)
				}

				buf.WriteTo(w)
				headerWritten = true
			}

			buf.Reset()
			err := fileTmpl.Execute(&buf, f)
			if err != nil {
				log.Printf("executing file template: %+v\n", err)
				if fatal {
					return
				}
			}

			buf.WriteTo(w)
		}
	}

	if !headerWritten {
		err := emptyHeaderTmpl.Execute(&buf, h)
		if err != nil {
			log.Fatalf("executing empty header template: %+v\n", err)
		}

		buf.WriteTo(w)
		headerWritten = true
	}
}

func processFile(name string, errChan chan<- error) <-chan file {
	fileChan := make(chan file)

	go func() {
		defer close(fileChan)

		var recursive bool
		if strings.HasSuffix(name, "/...") {
			recursive = true
			name = name[:len(name)-4]
		}

		stat, err := os.Stat(name)
		if err != nil {
			errChan <- errors.Wrap(err, "file info: "+name)
			return
		}

		if stat.IsDir() {
			if verbose {
				if recursive {
					log.Printf("walking directory '%s' recursively\n", name)
				} else {
					log.Printf("walking directory '%s'\n", name)
				}
			}
			filepath.Walk(name, func(path string, stat os.FileInfo, err error) error {
				if err != nil {
					if verbose {
						log.Printf("walking %s: %+v\n", path, err)
					}
					return filepath.SkipDir
				}

				if stat.IsDir() {
					if verbose {
						log.Printf("%s is a directory\n", path)
					}
					if !recursive && path != name {
						return filepath.SkipDir
					}

					return nil
				}

				if f, err := prepareFile(path, stat); err == nil {
					fileChan <- f
				} else {
					if verbose {
						log.Printf("prepare file %s: %+v\n", path, err)
					}
					return err
				}

				return nil
			})
		} else {
			if f, err := prepareFile(name, stat); err == nil {
				fileChan <- f
			} else {
				errChan <- err
			}
		}
	}()

	return fileChan
}

func prepareFile(name string, stat os.FileInfo) (file, error) {
	if verbose {
		log.Printf("preparing file '%s'\n", name)
	}
	if b, err := ioutil.ReadFile(name); err == nil {
		return file{
			name, fmt.Sprintf("%q", b), stat.Size(),
			uint32(stat.Mode()), stat.ModTime().Unix(),
		}, nil
	} else {
		return file{}, errors.Wrap(err, "reading file "+name)
	}
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\t%s [flags] files...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n\t\tAll arguments are expected to be files\n\t\t  or directories to be added to the output.\n\t\t  A directory suffixed by '...' will be added\n\t\t  recursively.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.StringVar(&output, "output", "file_data.go", "output file name. '-' for stdout")
	flag.StringVar(&input, "input", "", "input file name, to be used instead of the file arguments")
	flag.StringVar(&functionName, "function-name", "NewFileSystem", "name of the init function")
	flag.StringVar(&packageName, "package-name", "main", "package name of the generated file")
	flag.StringVar(&buildTags, "build-tags", "", "build tags for the generated file")
	flag.BoolVar(&fatal, "fatal-errors", false, "treat non-fatal errors as fatal")
	flag.BoolVar(&fallback, "fallback", false, "create an http.FileSystem that falls back to os.Open")
	flag.BoolVar(&verbose, "verbose", false, "output ")
}
