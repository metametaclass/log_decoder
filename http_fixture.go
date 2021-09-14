package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

// logLine represents log line with request or response data
type logLine struct {
	RequestID  string      `json:"request_id"`
	Message    string      `json:"msg"`
	Url        string      `json:"url"`
	Method     string      `json:"method"`
	BodyString string      `json:"body_string"`
	Headers    http.Header `json:"headers"`
	StatusCode int         `json:"status_code"`
	Status     string      `json:"status"`
}

// bodyWithHeaders represents request or response body and headers
type bodyWithHeaders struct {
	Headers    http.Header `json:"headers"`
	BodyString string      `json:"body_string,omitempty"`
	//BodyData   interface{} `json:"body_data"`
	BodyData interface{} `json:"-"`
}

// requestResponse represents request-response pair for fixture
type requestResponse struct {
	RequestID    string          `json:"request_id"`
	Url          string          `json:"url"`
	Method       string          `json:"method"`
	Request      bodyWithHeaders `json:"request"`
	SOAPRequest  *winrmRequest   `json:"soap_request"`
	StatusCode   int             `json:"status_code"`
	Status       string          `json:"status"`
	Response     bodyWithHeaders `json:"response"`
	SOAPResponse *winrmResponse  `json:"soap_response"`
}

type commandResponse struct {
	Command        string
	Response       interface{} `json:"response,omitempty"`
	ResponseString string      `json:"response_string,omitempty"`
}

// fixture is a list of request-response pairs
type fixture struct {
	data             []*requestResponse
	requestDict      map[string]*requestResponse
	commandResponses []*commandResponse
	currentCommand   string
}

func newFixture() *fixture {
	return &fixture{
		data:        []*requestResponse{},
		requestDict: make(map[string]*requestResponse),
	}
}

func (f *fixture) processLine(line []byte) {
	var l logLine
	err := json.Unmarshal(line, &l)
	if err != nil {
		return
	}
	if l.RequestID == "" {
		return
	}
	if l.BodyString == "" {
		return
	}
	switch l.Message {
	case "http_request":
		bodyString := ""
		n, err := decodeXML(l.BodyString)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parse %s xml request body in %+v\n", err, l)
			bodyString = l.BodyString
		}
		r, err := parseRequest(l.BodyString)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parse %s winrm soap request body in %+v\n", err, l)
		}

		pair := &requestResponse{
			RequestID: l.RequestID,
			Url:       l.Url,
			Method:    l.Method,
			Request: bodyWithHeaders{
				Headers:    l.Headers,
				BodyString: bodyString,
				BodyData:   n,
			},
			SOAPRequest: r,
		}
		f.requestDict[l.RequestID] = pair
		f.data = append(f.data, pair)
		f.processRequest(r)
	case "http_response":
		found, ok := f.requestDict[l.RequestID]
		if !ok {
			fmt.Fprintf(os.Stderr, "not found request %s for %+v", l.RequestID, l)
			return
		}
		n, err := decodeXML(l.BodyString)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid xml response body in %+v\n", l)
		}
		r, err := parseResponse(l.BodyString)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parse %s  winrm soap response body in %+v\n", err, l)
		}
		found.SOAPResponse = r
		found.Status = l.Status
		found.StatusCode = l.StatusCode
		found.Response = bodyWithHeaders{
			Headers:    l.Headers,
			BodyString: l.BodyString,
			BodyData:   n,
		}
		f.processResponse(r)
	}
}

func (f *fixture) processRequest(r *winrmRequest) {
	if r.Action == "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command" {
		if f.currentCommand != "" {
			fmt.Fprintf(os.Stderr, "Incorrect http fixture state on %s: %s\n", r.CommandKey, f.currentCommand)
		}
		f.currentCommand = r.CommandKey
	}
	if r.Action == "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Signal" {
		f.currentCommand = ""
	}
}

func (f *fixture) processResponse(r *winrmResponse) {
	if r.Action == "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/ReceiveResponse" {
		if f.currentCommand == "" {
			fmt.Fprintf(os.Stderr, "Incorrect http fixture state on %s\n", r.Action)
			return
		}
		responseString := r.CommandStdout
		if r.CommandStdoutJSON != nil {
			responseString = ""
		}
		f.commandResponses = append(f.commandResponses, &commandResponse{
			Command:        f.currentCommand,
			Response:       r.CommandStdoutJSON,
			ResponseString: responseString,
		})
	}
}

func (f *fixture) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(f.data, "", "  ")
	if err != nil {
		return errors.Wrap(err, "MarshalIndent failed")
	}
	err = ioutil.WriteFile(filename, data, 0660)
	if err != nil {
		return errors.Wrap(err, "WriteFile failed")
	}

	data, err = json.MarshalIndent(f.commandResponses, "", "  ")
	if err != nil {
		return errors.Wrap(err, "MarshalIndent failed")
	}
	err = ioutil.WriteFile(filename+"_responses.json", data, 0660)
	if err != nil {
		return errors.Wrap(err, "WriteFile failed")
	}

	return nil
}
