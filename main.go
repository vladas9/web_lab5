package main

import (
	"fmt"
	"os"
)

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
		fmt.Println("TODO: fetch", args[1])
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
