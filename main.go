package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func main() {
	filename := flag.String("filename", "", "filename to write decoded log")
	original := flag.String("original", "", "filename to write original log")
	skipFields := flag.String("skip", "", "list of fields to skip from dump")
	skipEmpty := flag.Bool("skipempty", false, "skip fields with empty values")
	flag.Parse()

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

	showI := func(name string, value interface{}) {
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
					return strings.Compare(sorted[i].k, sorted[j].k) < 0
				})

			}
			for _, v := range sorted {
				switch v.v.(type) {
				case map[string]interface{}:
					showI(v.k, v.v)
				case []interface{}:
					showI(v.k, v.v)
				default:
					fmt.Printf("%s: %+v\n", v.k, v.v)
					if writer != nil {
						_, err := fmt.Fprintf(writer, "%s: %+v\n", v.k, v.v)
						if err != nil {
							fmt.Fprintf(os.Stderr, "error write %s\n", err)
						}
					}
				}
			}
		}
		if !prevUnmarshalError {
			fmt.Println()
			if writer != nil {
				fmt.Fprintf(writer, "\n\n")
			}
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

func unmarshal(data []byte) (map[string]interface{}, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var linedata map[string]interface{}
	if err := d.Decode(&linedata); err != nil {
		return nil, err
	}
	return linedata, nil
}
