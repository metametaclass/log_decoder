package main

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/text/encoding/unicode"
)

type winrmRequest struct {
	Action        string `xml:"Header>Action"`
	ResourceURI   string `xml:"Header>ResourceURI" json:"ResourceURI,omitempty"`
	MessageID     string `xml:"Header>MessageID" json:"MessageID,omitempty"`
	RelatesTo     string `xml:"Header>RelatesTo" json:"RelatesTo,omitempty"`
	Selector      string `xml:"Header>SelectorSet>Selector" json:"Selector,omitempty"`
	Command       string `xml:"Body>CommandLine>Command" json:"Command,omitempty"`
	DesiredStream string `xml:"Body>Receive>DesiredStream" json:"DesiredStream,omitempty"`
	SignalCode    string `xml:"Body>Signal>Code" json:"SignalCode,omitempty"`
	PowerShell    bool   `json:"powershell"`
	CommandKey    string `json:"CommandKey,omitempty"`
}

type winrmResponse struct {
	Action            string           `xml:"Header>Action"`
	ResourceURI       string           `xml:"Header>ResourceURI" json:"ResourceURI,omitempty"`
	Selector          string           `xml:"Header>SelectorSet>Selector" json:"Selector,omitempty"`
	Stream            []responseStream `xml:"Body>ReceiveResponse>Stream" json:"Stream,omitempty"`
	ExitCode          string           `xml:"Body>ReceiveResponse>CommandState>ExitCode" json:"ExitCode,omitempty"`
	SignalResponse    string           `xml:"Body>SignalResponse" json:"SignalResponse,omitempty"`
	ShellID           string           `xml:"Body>Shell>ShellId" json:"ShellId,omitempty"`
	CommandStdout     string           `json:"command_stdout,omitempty"`
	CommandStdoutJSON interface{}      `json:"command_stdout_json,omitempty"`
	CommandStderr     string           `json:"command_stderr,omitempty"`
}

type responseStream struct {
	Name      string `xml:",attr"`
	CommandId string `xml:",attr"`
	End       bool   `xml:",attr" json:"End,omitempty"`
	Value     string `xml:",chardata"`
}

const powerShellCommandPrefix = "-EncodedCommand "

var urlRegexp = regexp.MustCompile(".*\\s-Uri\\s*\\\"(.*?)\\\"")

func parseRequest(body string) (*winrmRequest, error) {
	var r winrmRequest
	err := xml.Unmarshal([]byte(body), &r)
	if err != nil {
		return nil, err
	}
	r.PowerShell = strings.HasPrefix(strings.ToLower(r.Command), "powershell")
	if !r.PowerShell {
		r.CommandKey = r.Command
	} else {
		key, err := decodePowerShell(r.Command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "decodePowerShell error: %s\n", err)
		} else {
			r.CommandKey = key
		}
	}
	return &r, nil
}

func decodePowerShell(cmd string) (string, error) {
	i := strings.Index(cmd, powerShellCommandPrefix)
	if i < 0 {
		return "", errors.Errorf("Not fount powershell command prefix")
	}
	unicodeScript, err := base64.StdEncoding.DecodeString(cmd[i+len(powerShellCommandPrefix):])
	if err != nil {
		return "", errors.Wrap(err, "DecodeString failed")
	}

	decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
	b, err := decoder.Bytes(unicodeScript)
	if err != nil {
		return "", errors.Wrap(err, "decoder.Bytes failed")
	}
	script := string(b)

	ss := urlRegexp.FindStringSubmatch(script)
	// for i, s := range  {
	// 	fmt.Fprintf(os.Stderr, "%d:%s\n", i, s)
	// }
	if len(ss) < 2 {
		return "", errors.Errorf("Not found url in %s", script)
	}

	return ss[1], nil
}

func parseResponse(body string) (*winrmResponse, error) {
	var r winrmResponse
	err := xml.Unmarshal([]byte(body), &r)
	if err != nil {
		return nil, err
	}
	if len(r.Stream) > 0 {
		var stdout strings.Builder
		var stderr strings.Builder
		for _, s := range r.Stream {
			if s.Name == "stdout" {
				value, err := base64.StdEncoding.DecodeString(s.Value)
				if err != nil {
					fmt.Fprintf(os.Stderr, "DecodeString stdout failed at %+v %s\n", s, err)
				} else {
					stdout.Write(value)
				}
			}
			if s.Name == "stderr" {
				value, err := base64.StdEncoding.DecodeString(s.Value)
				if err != nil {
					fmt.Fprintf(os.Stderr, "DecodeString stderr failed at %+v %s\n", s, err)
				} else {
					stderr.Write(value)
				}

			}
		}
		var jsonData interface{}
		err := json.Unmarshal([]byte(stdout.String()), &jsonData)
		if err == nil {
			r.CommandStdoutJSON = jsonData
		} else {
			fmt.Fprintf(os.Stderr, "json.Unmarshal stdout failed: %s\n", err)
		}

		r.CommandStdout = stdout.String()
		r.CommandStderr = stderr.String()
	}
	return &r, nil
}
