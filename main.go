package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func main() {
	var writer io.Writer
	if len(os.Args) > 1 {
		filename := os.Args[1]
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE, 0660)
		if err != nil {
			fmt.Printf("error open file %s %s:", filename, err)
			os.Exit(1)
		}
		defer file.Close()
		writer = file
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {

		var linedata map[string]interface{}
		err := json.Unmarshal(scanner.Bytes(), &linedata)
		if err != nil {
			fmt.Printf("Unmarshal error %s\n", err)
			fmt.Printf("%s\n", scanner.Text())
			if writer != nil {
				fmt.Fprintf(writer, "Unmarshal error %s\n", err)
				fmt.Fprintf(writer, "%s\n", scanner.Text())
			}
		} else {
			type kv struct {
				k string
				v interface{}
			}
			sorted := make([]kv, 0)
			for k, v := range linedata {
				sorted = append(sorted, kv{k, v})
				sort.Slice(sorted, func(i, j int) bool {
					return strings.Compare(sorted[i].k, sorted[j].k) < 0
				})

			}
			for _, v := range sorted {
				fmt.Printf("%s: %+v\n", v.k, v.v)
				if writer != nil {
					fmt.Fprintf(writer, "%s: %+v\n", v.k, v.v)
				}
			}

		}
		fmt.Println()
		if writer != nil {
			fmt.Fprintf(writer, "\n\n")
		}
	}
}
