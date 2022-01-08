package main

import (
	"errors"
	"fmt"
	"strings"
)

// mergeErrors checks if any of errors is not nil and returns merged error
func mergeErrors(errs ...error) error {
	b := &strings.Builder{}
	for idx, err := range errs {
		if err != nil {
			if b.Len() > 0 {
				b.WriteString("; ")
			}
			_, _ = fmt.Fprintf(b, "error %d: %v", idx, err)
		}
	}
	if b.Len() > 0 {
		return errors.New(b.String())
	}
	return nil
}
