package main

import (
	"encoding/xml"
)

type winrmRequest struct {
	Action      string `xml:"Header>Action"`
	ResourceURI string `xml:"Header>ResourceURI" json:"ResourceURI,omitempty"`
	MessageID   string `xml:"Header>MessageID" json:"MessageID,omitempty"`
	RelatesTo   string `xml:"Header>RelatesTo" json:"RelatesTo,omitempty"`
	Selector    string `xml:"Header>SelectorSet>Selector" json:"Selector,omitempty"`
	//CommandLine string `xml:"Body>CommandLine>Command"`
	Command       string `xml:"Body>CommandLine>Command" json:"Command,omitempty"`
	DesiredStream string `xml:"Body>Receive>DesiredStream" json:"DesiredStream,omitempty"`
	SignalCode    string `xml:"Body>Signal>Code" json:"SignalCode,omitempty"`
}

type winrmResponse struct {
	Action         string           `xml:"Header>Action"`
	ResourceURI    string           `xml:"Header>ResourceURI" json:"ResourceURI,omitempty"`
	Selector       string           `xml:"Header>SelectorSet>Selector" json:"Selector,omitempty"`
	Stream         []responseStream `xml:"Body>ReceiveResponse>Stream" json:"Stream,omitempty"`
	ExitCode       string           `xml:"Body>CommandState>ExitCode" json:"ExitCode,omitempty"`
	SignalResponse string           `xml:"Body>SignalResponse" json:"SignalResponse,omitempty"`
	ShellID        string           `xml:"Body>Shell>ShellId" json:"ShellId,omitempty"`
}

type responseStream struct {
	Name      string `xml:",attr"`
	CommandId string `xml:",attr"`
	End       bool   `xml:",attr" json:"End,omitempty"`
	Value     string `xml:",chardata"`
}

func parseRequest(body string) (*winrmRequest, error) {
	var r winrmRequest
	err := xml.Unmarshal([]byte(body), &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func parseResponse(body string) (*winrmResponse, error) {
	var r winrmResponse
	err := xml.Unmarshal([]byte(body), &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
