package main

import (
	"fmt"
	"os"

	"github.com/ZyFoxX/ParaXm/pkg/cli"
    "github.com/ZyFoxX/ParaXm/pkg/scanner"
    "github.com/ZyFoxX/ParaXm/pkg/utils"
)

func main() {
	config, err := cli.ParseFlags()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if config.Help {
		cli.PrintUsage()
		return
	}

	if config.URLFile == "" && config.SingleURL == "" {
		fmt.Println("Error: Please provide either a URL (-u) or a file with URLs (-l)")
		cli.PrintUsage()
		return
	}

	var urls []string

	if config.URLFile != "" {
		var err error
		urls, err = utils.ReadLines(config.URLFile)
		if err != nil {
			fmt.Printf("Error reading URL file: %v\n", err)
			return
		}
	} else if config.SingleURL != "" {
		urls = []string{config.SingleURL}
	}

	fmt.Printf("[+] Starting ParaXm scan with %d threads\n", config.Threads)
	fmt.Printf("[+] Loaded %d URLs to scan\n", len(urls))
	fmt.Printf("[+] Request delay: %d second(s)\n", config.Delay)

	results := scanner.ScanURLs(urls, config.Threads, config.Timeout, config.Delay)
	
	for _, url := range results {
		fmt.Println(url)
	}
	
	if config.OutputFile != "" {
		utils.SaveResults(results, config.OutputFile)
		fmt.Printf("[+] Results saved to %s\n", config.OutputFile)
	}
	
	fmt.Printf("[+] Total URLs with parameters found: %d\n", len(results))
}
