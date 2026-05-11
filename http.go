package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       string
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

func fetchOnce(rawUrl string) (*Response, error) {
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

	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: go2web/1.0\r\nAccept: text/html,application/json\r\nConnection: close\r\n\r\n",
		u.RequestURI(), u.Hostname())

	return readConn(conn)
}

func fetch(rawUrl string) (*Response, error) {
	if resp, ok := cacheGet(rawUrl); ok {
		return resp, nil
	}

	current := rawUrl
	for range 10 {
		resp, err := fetchOnce(current)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 300 || resp.StatusCode >= 400 {
			if ttl := ttlFromHeaders(resp.Headers); ttl > 0 {
				cacheSet(rawUrl, resp, ttl)
			}
			return resp, nil
		}
		loc := resp.Headers["location"]
		if loc == "" {
			return resp, nil
		}
		base, err := url.Parse(current)
		if err != nil {
			return nil, err
		}
		target, err := url.Parse(loc)
		if err != nil {
			return nil, err
		}
		current = base.ResolveReference(target).String()
	}
	return nil, fmt.Errorf("too many redirects")
}
