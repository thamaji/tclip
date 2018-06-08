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
	fmt.Fprintln(output, "Usage: "+os.Args[0]+" [OPTIONS] FILE [FILE...]")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Table data to clipboard")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Options:")
	flag.CommandLine.PrintDefaults()
}

func convertToHTML(w io.Writer, input, format string) error {
	var file io.ReadCloser
	var r *csv.Reader
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

	r = csv.NewReader(file)
	var comma rune

	switch strings.ToLower(format) {
	case "tsv":
		comma = '\t'

	case "csv":
		comma = ','

	case "auto":
		if input != "-" {
			switch strings.ToLower(filepath.Ext(input)) {
			case ".tsv":
				comma = '\t'

			case ".csv":
				comma = ','
			}
		}

		if comma != 0 {
			break
		}

		buf := bytes.NewBuffer(make([]byte, 0, 1024))
		if _, err := io.Copy(buf, file); err != nil {
			return err
		}

		var max int
		for _, c := range []rune{'\t', ','} {
			var record []string
			var err error
			r := csv.NewReader(bytes.NewReader(buf.Bytes()))
			r.Comma = c
			r.LazyQuotes = true
			r.FieldsPerRecord = -1
			fields := 0
			for {
				record, err = r.Read()
				if record != nil {
					fields += len(record)
				}
				if err != nil {
					break
				}
			}

			if err != io.EOF {
				continue
			}

			if max < fields {
				max = fields
				comma = c
			}
		}

		r = csv.NewReader(bytes.NewReader(buf.Bytes()))

	default:
		return errors.New("unsupported format: " + format)
	}

	if comma == 0 {
		return errors.New("unknown format")
	}

	r.Comma = comma
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	for {
		record, err := r.Read()
		if record != nil {
			fmt.Fprint(w, "<tr>")
			for _, field := range record {
				fmt.Fprint(w, "<td>"+html.EscapeString(field)+"</td>")
			}
			fmt.Fprint(w, "</tr>")
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	flag.Usage = usage
	output := flag.CommandLine.Output()

	var format string
	var join bool
	var version, help bool

	flag.StringVar(&format, "f", "auto", "set input table format; formats are tsv,csv,auto")
	flag.BoolVar(&join, "j", false, "join multi tables")
	flag.BoolVar(&version, "v", false, "show version")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()

	if help {
		usage()
		return
	}

	if version {
		fmt.Fprintln(output, "1.0.3")
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

	if join {
		fmt.Fprint(w, "<table>")
		for _, arg := range args {
			if err := convertToHTML(w, arg, format); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				return
			}
		}
		fmt.Fprint(w, "</table>")

	} else {
		for _, arg := range args {
			fmt.Fprint(w, "<table>")
			if err := convertToHTML(w, arg, format); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				return
			}
			fmt.Fprint(w, "</table>")
		}
	}

	w.Close()

	if err := cmd.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
