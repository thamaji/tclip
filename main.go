package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func usage() {
	output := flag.CommandLine.Output()
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Usage: "+os.Args[0]+" FILE [FILE...]")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Table data to clipboard")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Options:")
	flag.CommandLine.PrintDefaults()
}

func convertToHTML(w io.Writer, input, format string) error {
	var file io.ReadCloser
	var err error

	if input == "-" {
		file = os.Stdin

	} else {
		file, err = os.Open(input)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	r := csv.NewReader(file)

	switch strings.ToLower(format) {
	case "tsv":
		r.Comma = '\t'

	case "csv":
		r.Comma = ','

	case "auto":
		if input != "" {
			switch strings.ToLower(filepath.Ext(input)) {
			case ".tsv":
				r.Comma = '\t'

			case ".csv":
				r.Comma = ','

			default:
				rbuf := bytes.NewBuffer(make([]byte, 0, 1024))
				if _, err := io.Copy(rbuf, file); err != nil {
					return err
				}
				buf := rbuf.Bytes()

				r1 := csv.NewReader(bytes.NewReader(buf))
				r1.Comma = '\t'
				if _, err := r.ReadAll(); err == nil {
					r = csv.NewReader(bytes.NewReader(buf))
					r.Comma = r1.Comma
					break
				}

				r1 = csv.NewReader(bytes.NewReader(buf))
				r1.Comma = ','
				if _, err := r.ReadAll(); err == nil {
					r = csv.NewReader(bytes.NewReader(buf))
					r.Comma = r1.Comma
					break
				}

				return errors.New("unknown format")
			}
		}

	default:
		return errors.New("unsupported format: " + format)
	}

	r.Comma = '\t'
	r.LazyQuotes = true

	fmt.Fprint(w, "<table>")
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fmt.Fprint(w, "<tr>")
		for _, field := range record {
			fmt.Fprint(w, "<td>"+html.EscapeString(field)+"</td>")
		}
		fmt.Fprint(w, "</tr>")
	}
	fmt.Fprint(w, "</table>")

	return nil
}

func main() {
	flag.Usage = usage
	output := flag.CommandLine.Output()

	var format string
	var version, help bool

	flag.StringVar(&format, "f", "auto", "set input table format; formats are tsv,csv,auto")
	flag.BoolVar(&version, "v", false, "show version")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()

	if help {
		usage()
		return
	}

	if version {
		fmt.Fprintln(output, "1.0.1")
		return
	}

	args := flag.Args()
	if len(args) <= 0 {
		args = append(args, "-")
	}

	cmd := exec.Command("xclip", "-t", "text/html", "-selection", "clipboard", "-i")
	w, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	for _, arg := range args {
		if err := convertToHTML(w, arg, format); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}
	}

	w.Close()

	if err := cmd.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
