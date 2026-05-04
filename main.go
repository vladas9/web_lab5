package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       string
}

type SearchResult struct {
	Title string
	URL   string
}

func readChunked(reader *bufio.Reader) ([]byte, error) {
	var body []byte
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		size, err := strconv.ParseInt(strings.TrimSpace(line), 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid chunk size: %s", strings.TrimSpace(line))
		}
		if size == 0 {
			break
		}
		chunk := make([]byte, size)
		if _, err = io.ReadFull(reader, chunk); err != nil {
			return nil, err
		}
		body = append(body, chunk...)
		reader.ReadString('\n') // consume trailing CRLF
	}
	return body, nil
}

func readConn(conn net.Conn) (*Response, error) {
	reader := bufio.NewReader(conn)

	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(strings.TrimSpace(statusLine), " ", 3)
	code, _ := strconv.Atoi(parts[1])

	headers := make(map[string]string)
	for {
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		idx := strings.Index(line, ":")
		if idx > 0 {
			k := strings.ToLower(strings.TrimSpace(line[:idx]))
			v := strings.TrimSpace(line[idx+1:])
			headers[k] = v
		}
	}

	var body []byte
	if strings.Contains(headers["transfer-encoding"], "chunked") {
		body, err = readChunked(reader)
	} else {
		body, err = io.ReadAll(reader)
	}
	if err != nil {
		return nil, err
	}

	return &Response{StatusCode: code, Headers: headers, Body: string(body)}, nil
}

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

func parseHTML(body string) string {
	doc, _ := html.Parse(strings.NewReader(body))
	return extractText(doc)
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

func fetch(rawUrl string) (*Response, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	ports := map[string]string{"http": "80", "https": "443"}
	port, ok := ports[u.Scheme]
	if !ok {
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
	addr := u.Hostname() + ":" + port

	var conn net.Conn
	if u.Scheme == "https" {
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: u.Hostname()})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: go2web/1.0\r\nConnection: close\r\n\r\n",
		u.RequestURI(), u.Hostname())

	return readConn(conn)
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

func printHelp() {
	fmt.Println(`go2web - HTTP over TCP sockets

Usage:
  go2web -u <URL>          Make an HTTP request and print the response
  go2web -s <search-term>  Search DuckDuckGo and print top 10 results
  go2web -h                Show this help`)
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}

	switch args[0] {
	case "-h", "--help":
		printHelp()
	case "-u":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: -u requires a URL")
			os.Exit(1)
		}
		resp, err := fetch(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println(parseHTML(resp.Body))
	case "-s":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: -s requires a search term")
			os.Exit(1)
		}
		term := strings.Join(args[1:], " ")
		if err := search(term); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[0])
		os.Exit(1)
	}
}
