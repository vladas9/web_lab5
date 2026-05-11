package main

import (
	"encoding/json"
	"strings"

	"golang.org/x/net/html"
)

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	result := ""
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += extractText(c)
	}
	return result
}

func hasClass(n *html.Node, cls string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == cls {
					return true
				}
			}
		}
	}
	return false
}

func parseHTML(body string) string {
	doc, _ := html.Parse(strings.NewReader(body))
	return extractText(doc)
}

func formatBody(resp *Response) string {
	ct := resp.Headers["content-type"]
	if strings.Contains(ct, "application/json") {
		var v any
		if err := json.Unmarshal([]byte(resp.Body), &v); err == nil {
			if pretty, err := json.MarshalIndent(v, "", "  "); err == nil {
				return string(pretty)
			}
		}
		return resp.Body
	}
	return parseHTML(resp.Body)
}
