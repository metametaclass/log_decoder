package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// logWriter contains all output writers for decoded logs
type logWriter struct {
	decodedWriter  io.WriteCloser
	originalWriter io.WriteCloser
	errorWriter    io.WriteCloser
}

func (w *logWriter) Close() error {
	var errDecodedWriter error
	if w.decodedWriter != nil {
		errDecodedWriter = w.decodedWriter.Close()
	}
	var errOriginalWriter error
	if w.originalWriter != nil {
		errOriginalWriter = w.originalWriter.Close()
	}
	var errErrorWriter error
	if w.errorWriter != nil {
		errErrorWriter = w.errorWriter.Close()
	}
	return mergeErrors(errDecodedWriter, errOriginalWriter, errErrorWriter)
}

func newWriter() *logWriter {
	return &logWriter{}
}

// openLogFile opens file and stores it to reference to interface
func openLogFile(filename string, fileRef *io.WriteCloser) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		return errors.Wrapf(err, "OpenFile %s failed", filename)
	}
	*fileRef = file
	return nil
}

func (w *logWriter) OpenDecoded(filename string) error {
	return openLogFile(filename, &w.decodedWriter)
}

func (w *logWriter) OpenError(filename string) error {
	return openLogFile(filename, &w.errorWriter)
}

func (w *logWriter) OpenOriginal(filename string) error {
	return openLogFile(filename, &w.originalWriter)
}

func (w *logWriter) OpenAll(decodedFilename, errorFilename, originalFilename string) error {
	if decodedFilename != "" {
		err := w.OpenDecoded(decodedFilename)
		if err != nil {
			return err
		}
	}
	if errorFilename != "" {
		err := w.OpenError(errorFilename)
		if err != nil {
			return err
		}
	}
	if originalFilename != "" {
		err := w.OpenOriginal(originalFilename)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *logWriter) OpenWithPrefix(prefix string) error {
	err := w.OpenDecoded(fmt.Sprintf("%s_log_decoded.txt", prefix))
	if err != nil {
		return err
	}
	err = w.OpenError(fmt.Sprintf("%s_log_error.txt", prefix))
	if err != nil {
		return err
	}
	err = w.OpenOriginal(fmt.Sprintf("%s_log_original.txt", prefix))
	if err != nil {
		return err
	}
	return nil
}

func (w *logWriter) WriteOriginal(b []byte) {
	if w.originalWriter == nil {
		return
	}
	_, err := w.originalWriter.Write(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WriteOriginal error %s\n", err)
	}
	fmt.Fprintf(w.originalWriter, "\n")

}

func (w *logWriter) WriteText(text string) {
	fmt.Printf("%s\n", text)
	if w.decodedWriter != nil {
		fmt.Fprintf(w.decodedWriter, "%s\n", text)
	}
}

func (w *logWriter) WriteTextAndError(comment, text string, err error) {
	fmt.Printf("%s error %s\n%s\n", comment, err, text)
	if w.decodedWriter != nil {
		fmt.Fprintf(w.decodedWriter, "%s error %s\n%s\n", comment, err, text)
	}
}

func (w *logWriter) WriteIface(level logLevel, name string, value interface{}) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Marshal error %s\n", err)
		return
	}
	fmt.Printf("%s: %s\n", name, string(b))
	if w.decodedWriter != nil {
		_, err := fmt.Fprintf(w.decodedWriter, "%s: %s\n", name, string(b))
		if err != nil {
			fmt.Fprintf(os.Stderr, "WriteIface: error write %s\n", err)
		}
	}
	if level.IsErrorOrWarn() && w.errorWriter != nil {
		_, err := fmt.Fprintf(w.errorWriter, "%s: %s\n", name, string(b))
		if err != nil {
			fmt.Fprintf(os.Stderr, "WriteIface: error write %s\n", err)
		}
	}
}

func (w *logWriter) WriteValue(level logLevel, name string, value interface{}) {
	s := fmt.Sprintf("%+v", value)
	// s = strings.TrimSpace(s)
	// s = strings.Replace(s, "\n\n", "\\n\n", -1)
	// s = strings.Replace(s, "\r\n\r\n", "\\r\\n\n", -1)
	if strings.Contains(s, "\n") {
		s = strings.Replace(s, "\n", "\n\t\t", -1)
		s = fmt.Sprintf("| \n\t\t%s", s)
	}

	fmt.Printf("%s: %s\n", name, s)
	if w.decodedWriter != nil {
		_, err := fmt.Fprintf(w.decodedWriter, "%s: %s\n", name, s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error write %s\n", err)
		}
	}
	if level.IsErrorOrWarn() && w.errorWriter != nil {
		_, err := fmt.Fprintf(w.errorWriter, "%s: %s\n", name, s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error write %s\n", err)
		}
	}
}

func (w *logWriter) WriteNewLine(level logLevel) {
	fmt.Println()
	if w.decodedWriter != nil {
		fmt.Fprintf(w.decodedWriter, "\n\n")
	}
	if level.IsErrorOrWarn() && w.errorWriter != nil {
		fmt.Fprintf(w.errorWriter, "\n\n")
	}
}
