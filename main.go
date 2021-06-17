package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {

		var linedata map[string]interface{}
		err := json.Unmarshal(scanner.Bytes(), &linedata)
		if err != nil {
			fmt.Printf("Unmarshal error %s\n%s\n", err, scanner.Text())
		} else {
			for k, v := range linedata {
				fmt.Printf("%s: %+v\n", k, v)
			}
		}
	}
}
