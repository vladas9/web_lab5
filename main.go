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
)

type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       string
}

func fetch(rawUrl string) (*Response, error) {
	url, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	ports := map[string]string{"http": "80", "https": "443"}
	port, ok := ports[url.Scheme]
	if !ok {
		return nil, fmt.Errorf("unsupported scheme: %s", url.Scheme)
	}
	addr := url.Hostname() + ":" + port

	var conn net.Conn
	if url.Scheme == "https" {
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: url.Hostname()})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n",
		url.RequestURI(), url.Hostname())

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

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return &Response{StatusCode: code, Headers: headers, Body: string(body)}, nil
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
		fmt.Printf("Status: %d\n\n", resp.StatusCode)
		fmt.Println(resp)
	case "-s":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: -s requires a search term")
			os.Exit(1)
		}
		fmt.Println("TODO: search", args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[0])
		os.Exit(1)
	}
}
