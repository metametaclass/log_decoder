package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type showFn func(isErrorOrWarn bool, name string, value interface{})

var wellKnownFields = map[string]int{
	"time":           1,
	"caller":         2,
	"level":          3,
	"msg":            4,
	"error":          5,
	"error_verbose":  6,
	"trace_id":       7,
	"request_id":     8,
	"int_request_id": 9,
	"pid":            10,
	"version":        11,
}

func main() {
	filename := flag.String("filename", "", "filename to write decoded log")
	errorFilename := flag.String("error", "", "filename to write decoded error log")
	fixtureFile := flag.String("fixture", "", "filename to write request->response fixture")
	original := flag.String("original", "", "filename to write original log")
	skipFields := flag.String("skip", "", "list of fields to skip from dump")
	skipEmpty := flag.Bool("skipempty", false, "skip fields with empty values")
	flag.Parse()

	fixture := newFixture()

	var writer io.Writer
	if *filename != "" {
		file, err := os.OpenFile(*filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
		if err != nil {
			fmt.Printf("error open file %s %s:", *filename, err)
			os.Exit(1)
		}
		defer file.Close()
		writer = file
	}

	var errorWriter io.Writer
	if *errorFilename != "" {
		file, err := os.OpenFile(*errorFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
		if err != nil {
			fmt.Printf("error open file %s %s:", *filename, err)
			os.Exit(1)
		}
		defer file.Close()
		errorWriter = file
	}

	var originalWriter io.Writer
	if *original != "" {
		originalFile, err := os.OpenFile(*original, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
		if err != nil {
			fmt.Printf("error open file %s %s:", *original, err)
			os.Exit(1)
		}
		defer originalFile.Close()
		originalWriter = originalFile
	}

	showI := func(isErrorOrWarn bool, name string, value interface{}) {
		b, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			fmt.Fprintf(writer, "Marshal error %s\n", err)
			return
		}
		fmt.Printf("%s: %s\n", name, string(b))
		if writer != nil {
			_, err := fmt.Fprintf(writer, "%s: %s\n", name, string(b))
			if err != nil {
				fmt.Fprintf(os.Stderr, "showI: error write %s\n", err)
			}
		}
		if isErrorOrWarn && errorWriter != nil {
			_, err := fmt.Fprintf(errorWriter, "%s: %s\n", name, string(b))
			if err != nil {
				fmt.Fprintf(os.Stderr, "showI: error write %s\n", err)
			}
		}
	}

	showV := func(isErrorOrWarn bool, name string, value interface{}) {
		s := fmt.Sprintf("%+v", value)
		// s = strings.TrimSpace(s)
		// s = strings.Replace(s, "\n\n", "\\n\n", -1)
		// s = strings.Replace(s, "\r\n\r\n", "\\r\\n\n", -1)
		if strings.Contains(s, "\n") {
			s = strings.Replace(s, "\n", "\n\t\t", -1)
			s = fmt.Sprintf("| \n\t\t%s", s)
		}

		fmt.Printf("%s: %s\n", name, s)
		if writer != nil {
			_, err := fmt.Fprintf(writer, "%s: %s\n", name, s)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error write %s\n", err)
			}
		}
		if isErrorOrWarn && errorWriter != nil {
			_, err := fmt.Fprintf(errorWriter, "%s: %s\n", name, s)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error write %s\n", err)
			}
		}
	}

	skipFieldsMap := make(map[string]struct{})
	if *skipFields != "" {
		for _, key := range strings.Split(*skipFields, ",") {
			skipFieldsMap[key] = struct{}{}
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	buffer := make([]byte, 0, 262144)
	scanner.Buffer(buffer, 524288)
	prevUnmarshalError := false
	for scanner.Scan() {
		if originalWriter != nil {
			_, err := originalWriter.Write(scanner.Bytes())
			if err != nil {
				fmt.Printf("originalWriter.Write error %s\n", err)
			}
			fmt.Fprintf(originalWriter, "\n")
		}
		fixture.processLine(scanner.Bytes())

		var isErrorOrWarn bool
		linedata, err := unmarshal(scanner.Bytes())
		if err != nil {
			text := strings.Trim(scanner.Text(), "\r\n")
			if prevUnmarshalError {
				fmt.Printf("%s\n", text)
				if writer != nil {
					fmt.Fprintf(writer, "%s\n", text)
				}
			} else {
				fmt.Printf("Unmarshal error %s\n%s\n", err, text)
				if writer != nil {
					fmt.Fprintf(writer, "Unmarshal error %s\n%s\n", err, text)
				}
			}
			prevUnmarshalError = true
		} else {
			var level string
			levelIface := linedata["level"]
			if levelIface != nil {
				level, _ = levelIface.(string)
			}
			isErrorOrWarn = level == "error" || level == "warn"

			prevUnmarshalError = false
			type kv struct {
				k string
				v interface{}
			}
			sorted := make([]kv, 0)
			for k, v := range linedata {
				if _, skip := skipFieldsMap[k]; skip {
					continue
				}
				if *skipEmpty && isEmpty(v) {
					continue
				}
				sorted = append(sorted, kv{k, v})
				sort.Slice(sorted, func(i, j int) bool {
					wellKnown1, ok1 := wellKnownFields[sorted[i].k]
					wellKnown2, ok2 := wellKnownFields[sorted[j].k]
					if ok1 && ok2 {
						return wellKnown1 < wellKnown2
					}
					if ok1 && !ok2 {
						return true
					}
					if !ok1 && ok2 {
						return false
					}
					return strings.Compare(sorted[i].k, sorted[j].k) < 0
				})
			}
			for _, v := range sorted {
				if v.k == "body_string" {
					showXMLBody(showI, v.k, v.v)
				}
				switch v.v.(type) {
				case map[string]interface{}:
					showI(isErrorOrWarn, v.k, v.v)
				case []interface{}:
					showI(isErrorOrWarn, v.k, v.v)
				default:
					showV(isErrorOrWarn, v.k, v.v)
				}
			}
		}
		if !prevUnmarshalError {
			fmt.Println()
			if writer != nil {
				fmt.Fprintf(writer, "\n\n")
			}
			if isErrorOrWarn && errorWriter != nil {
				fmt.Fprintf(errorWriter, "\n\n")
			}
		}
	}

	if *fixtureFile != "" {
		err := fixture.SaveToFile(*fixtureFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "SaveToFile error %s\n", err)
		}
	}
}

func isEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	switch v.(type) {
	case string:
		return v == ""
	case float32:
		return v == 0.0
	case float64:
		return v == 0.0
	case int:
		return v == 0
	case int64:
		return v == 0
	default:
		return false
	}
}

func showXMLBody(show showFn, name string, value interface{}) {
	str, ok := value.(string)
	if !ok {
		fmt.Fprintf(os.Stderr, "invalid body string xml")
		return
	}
	n, err := decodeXML(str)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid body string xml: %s", err)
		return
	}
	show(false, name+"_xml_json", n)
	data, err := xml.MarshalIndent(n, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "xml.MarshalIndent error: %s", err)
		return
	}
	fmt.Printf("%s_xml: %s\n", name, string(data))
}

func unmarshal(data []byte) (map[string]interface{}, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var linedata map[string]interface{}
	if err := d.Decode(&linedata); err != nil {
		return nil, err
	}
	return linedata, nil
}
