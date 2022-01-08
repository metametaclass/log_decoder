package main

import (
	"encoding/xml"
	"strings"
)

type Node struct {
	XMLName    xml.Name
	Attributes []xml.Attr `xml:",any,attr" json:"attrs,omitempty"`
	Nodes      []Node     `xml:",any" json:"nodes,omitempty"`
	CharData   string     `xml:",chardata" json:"text,omitempty"`
}

// func (n *Node) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
// 	//n.Attrs = start.Attr
// 	type node Node

// 	return d.DecodeElement((*node)(n), &start)
// }

func walk(offset int, nodes []Node, f func(int, Node) bool) {
	for _, n := range nodes {
		if f(offset, n) {
			walk(offset+2, n.Nodes, f)
		}
	}
}

func decodeXML(data string) (*Node, error) {
	dec := xml.NewDecoder(strings.NewReader(data))
	var n Node
	err := dec.Decode(&n)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// func main() {
//

//     var n Node
//     err := dec.Decode(&n)
//     if err != nil {
//         panic(err)
//     }

//     dataXml, err := xml.Marshal(n)
//     if err != nil {
//         panic(err)
//     }
//     fmt.Println(string(dataXml))

//     dataXml1, err := xml.MarshalIndent(n, "", "  ")
//     if err != nil {
//         panic(err)
//     }
//     fmt.Println(string(dataXml1))

//     data, err := json.MarshalIndent(n, "", "  ")
//     if err != nil {
//         panic(err)
//     }
//     fmt.Println(string(data))

//     walk(0, []Node{n}, func(offset int, n Node) bool {
//         //fmt.Println(string(n.Content))
//         //fmt.Println(n.Attrs)
//         s := strings.Repeat(" ", offset)
//         fmt.Printf("%s%s\n", s, n.XMLName)
//         // if len(n.Content) > 0 {
//         //  fmt.Printf("%s  [%s]\n", s, n.Content)
//         // }
//         for _, attr := range n.Attrs {
//             fmt.Printf("%s  %s=%s\n", s, attr.Name, attr.Value)

//         }
//         if len(n.CharData) > 0 {
//             fmt.Printf("%s  [%s]\n", s, n.CharData)
//         }
//         return true
//     })
// }
