package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// logWriter contains all output writers for decoded logs
type logWriter struct {
	needColors        bool
	warnColor         string
	resetColor        string
	hideDebug         bool
	decodedWriter     io.WriteCloser
	decodedInfoWriter io.WriteCloser
	originalWriter    io.WriteCloser
	errorWriter       io.WriteCloser
}

func (w *logWriter) Close() error {
	var errDecodedWriter error
	if w.decodedWriter != nil {
		errDecodedWriter = w.decodedWriter.Close()
	}
	var errDecodedInfoWriter error
	if w.decodedInfoWriter != nil {
		errDecodedInfoWriter = w.decodedInfoWriter.Close()
	}
	var errOriginalWriter error
	if w.originalWriter != nil {
		errOriginalWriter = w.originalWriter.Close()
	}
	var errErrorWriter error
	if w.errorWriter != nil {
		errErrorWriter = w.errorWriter.Close()
	}
	return mergeErrors(errDecodedWriter, errDecodedInfoWriter, errOriginalWriter, errErrorWriter)
}

func newWriter(hideDebug bool) *logWriter {
	needColors := runtime.GOOS == "linux" || runtime.GOOS == "darwin"
	warnColor := ""
	resetColor := ""
	if needColors {
		warnColor = levelToColor(logLevelWarn)
		resetColor = "\u001b[0m"
	}
	return &logWriter{
		needColors: needColors,
		warnColor:  warnColor,
		resetColor: resetColor,
		hideDebug:  hideDebug,
	}
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

func (w *logWriter) OpenDecodedInfo(filename string) error {
	return openLogFile(filename, &w.decodedInfoWriter)
}

func (w *logWriter) OpenError(filename string) error {
	return openLogFile(filename, &w.errorWriter)
}

func (w *logWriter) OpenOriginal(filename string) error {
	return openLogFile(filename, &w.originalWriter)
}

func (w *logWriter) OpenAll(decodedFilename, decodedInfoFilename, errorFilename, originalFilename string) error {
	if decodedFilename != "" {
		err := w.OpenDecoded(decodedFilename)
		if err != nil {
			return err
		}
	}
	if decodedInfoFilename != "" {
		err := w.OpenDecoded(decodedInfoFilename)
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
	err := w.OpenDecoded(fmt.Sprintf("%s_log_decoded.log", prefix))
	if err != nil {
		return err
	}
	err = w.OpenDecodedInfo(fmt.Sprintf("%s_log_info.log", prefix))
	if err != nil {
		return err
	}
	err = w.OpenError(fmt.Sprintf("%s_log_error.log", prefix))
	if err != nil {
		return err
	}
	err = w.OpenOriginal(fmt.Sprintf("%s_log_original.log", prefix))
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
	fmt.Printf("%s%s error %s\n%s%s\n", w.warnColor, comment, err, text, w.resetColor)
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
	if !w.hideDebug || level.IsInfoOrHigher() {
		color := ""
		if w.needColors {
			color = levelToColor(level)
		}
		fmt.Printf("%s%s: %s%s\n", color, name, string(b), w.resetColor)
	}
	if w.decodedWriter != nil {
		_, err := fmt.Fprintf(w.decodedWriter, "%s: %s\n", name, string(b))
		if err != nil {
			fmt.Fprintf(os.Stderr, "WriteIface: error write %s\n", err)
		}
	}
	if level.IsInfoOrHigher() && w.decodedInfoWriter != nil {
		_, err := fmt.Fprintf(w.decodedInfoWriter, "%s: %s\n", name, string(b))
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

	if !w.hideDebug || level.IsInfoOrHigher() {
		color := ""
		if w.needColors {
			color = levelToColor(level)
		}

		fmt.Printf("%s%s: %s%s\n", color, name, s, w.resetColor)
	}
	if w.decodedWriter != nil {
		_, err := fmt.Fprintf(w.decodedWriter, "%s: %s\n", name, s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error write %s\n", err)
		}
	}
	if level.IsInfoOrHigher() && w.decodedInfoWriter != nil {
		_, err := fmt.Fprintf(w.decodedInfoWriter, "%s: %s\n", name, s)
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
	if !w.hideDebug || level.IsInfoOrHigher() {
		fmt.Println()
	}
	if w.decodedWriter != nil {
		fmt.Fprintf(w.decodedWriter, "\n\n")
	}
	if level.IsInfoOrHigher() && w.decodedInfoWriter != nil {
		fmt.Fprintf(w.decodedInfoWriter, "\n\n")
	}
	if level.IsErrorOrWarn() && w.errorWriter != nil {
		fmt.Fprintf(w.errorWriter, "\n\n")
	}
}
