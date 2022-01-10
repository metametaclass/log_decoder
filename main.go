package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

type showFn func(level logLevel, name string, value interface{})

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
	prefix := flag.String("prefix", "", "filename prefix for all logs")
	skipFields := flag.String("skip", "", "list of fields to skip from dump")
	skipEmpty := flag.Bool("skipempty", false, "skip fields with empty values")
	flag.Parse()

	fixture := newFixture()
	writer := newWriter()
	defer writer.Close()

	if *prefix != "" {
		err := writer.OpenWithPrefix(*prefix)
		if err != nil {
			fmt.Printf("OpenWithPrefix error %s %s:", *prefix, err)
			os.Exit(1)
		}
	} else {
		err := writer.OpenAll(*filename, *errorFilename, *original)
		if err != nil {
			fmt.Printf("OpenAll error %s %s:", *filename, err)
			os.Exit(1)
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
	scanner.Buffer(buffer, 32*1048576)
	prevUnmarshalError := false
	for scanner.Scan() {
		writer.WriteOriginal(scanner.Bytes())
		fixture.processLine(scanner.Bytes())

		var logLevel logLevel
		linedata, err := unmarshal(scanner.Bytes())
		if err != nil {
			text := strings.Trim(scanner.Text(), "\r\n")
			if prevUnmarshalError {
				writer.WriteText(text)
			} else {
				writer.WriteTextAndError("Unmarshal", text, err)
			}
			prevUnmarshalError = true
		} else {
			var level string
			levelIface := linedata["level"]
			if levelIface != nil {
				level, _ = levelIface.(string)
			}
			logLevel := parseLogLevel(level)

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
					showXMLBody(writer.WriteIface, v.k, v.v)
				}
				switch v.v.(type) {
				case map[string]interface{}:
					writer.WriteIface(logLevel, v.k, v.v)
				case []interface{}:
					writer.WriteIface(logLevel, v.k, v.v)
				default:
					writer.WriteValue(logLevel, v.k, v.v)
				}
			}
		}
		if !prevUnmarshalError {
			writer.WriteNewLine(logLevel)
		}
	}
	if scanner.Err() != nil {
		writer.WriteTextAndError("scanner error", "", scanner.Err())
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
	show(logLevelInfo, name+"_xml_json", n)
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
