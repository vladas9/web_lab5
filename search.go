package main

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type SearchResult struct {
	Title string
	URL   string
}

func extractSearchResults(body string) []SearchResult {
	doc, _ := html.Parse(strings.NewReader(body))
	var results []SearchResult

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if len(results) >= 10 {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" && hasClass(n, "result__a") {
			title := strings.TrimSpace(extractText(n))
			for _, a := range n.Attr {
				if a.Key == "href" {
					u, err := url.Parse(a.Val)
					resultURL := a.Val
					if err == nil {
						if uddg := u.Query().Get("uddg"); uddg != "" {
							resultURL, _ = url.QueryUnescape(uddg)
						}
					}
					if title != "" && resultURL != "" {
						results = append(results, SearchResult{Title: title, URL: resultURL})
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)
	return results
}

func search(term string) error {
	query := url.QueryEscape(term)
	resp, err := fetch("https://html.duckduckgo.com/html/?q=" + query)
	if err != nil {
		return err
	}

	results := extractSearchResults(resp.Body)
	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	for i, r := range results {
		fmt.Printf("%d. %s\n   %s\n\n", i+1, r.Title, r.URL)
	}
	return nil
}
