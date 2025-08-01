package cli

import (
	"flag"
	"fmt"
)

type Config struct {
	URLFile    string
	SingleURL  string
	OutputFile string
	Threads    int
	Timeout    int
	Delay      int
	Help       bool
}

func ParseFlags() (Config, error) {
	config := Config{}
	
	flag.StringVar(&config.URLFile, "l", "", "File containing URLs to scan")
	flag.StringVar(&config.SingleURL, "u", "", "Single URL to scan")
	flag.StringVar(&config.OutputFile, "o", "", "Output file to write results")
	flag.IntVar(&config.Threads, "t", 10, "Number of concurrent threads")
	flag.IntVar(&config.Timeout, "timeout", 15, "Timeout in seconds for HTTP requests")
	flag.IntVar(&config.Delay, "d", 1, "Delay between requests in seconds")
	flag.BoolVar(&config.Help, "h", false, "Show help")
	
	flag.Parse()
	
	return config, nil
}

func PrintUsage() {
	fmt.Println(`
ParaXm - URL Params Finder

Usage:
  ./ParaXm [options]

Options:
  -u string     Single URL to scan
  -l string     File containing URLs to scan
  -o string     Output file to write results
  -t int        Number of concurrent threads (default 10)
  -timeout int  Timeout in seconds for HTTP requests (default 15)
  -d int        Delay between requests in seconds (default 1)
  -h            Show help

Examples:
  ./ParaXm -u https://example.com
  ./ParaXm -l urls.txt -o results.txt
  ./ParaXm -u https://example.com -t 20 -d 2
`)
}