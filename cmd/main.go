package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ParaXm/pkg"
)

const (
	VERSION  = "1.0"
	CODENAME = "URL Parameter Hunter - github.com/ZyFoxX"
)

var banner = `
__________                             ____  ___          
\______   \ _____    _______  _____    \   \/  /   _____  
 |     ___/ \__  \   \_  __ \ \__  \    \     /   /     \ 
 |    |      / __ \_  |  | \/  / __ \_  /     \  |  Y Y  \
 |____|     (____  /  |__|    (____  / /___/\  \ |__|_|  /
                 \/                \/        \_/       \/`

func printBanner() {
	fmt.Println(banner)
	fmt.Printf("\nParaXm v%s - %s\n\n", VERSION, CODENAME)
}

func printUsage() {
	fmt.Print("Usage: paraxm [options]\n\n")
	fmt.Print("Options:\n")
	fmt.Print("  -u             Target URL to scan (e.g., https://example.com)\n")
	fmt.Print("  -o             Output file to save results\n")
	fmt.Print("  -d             Crawling depth (default: 2)\n")
	fmt.Print("  -t             Number of concurrent threads (default: 10)\n")
	fmt.Print("  -ratelimit     Maximum requests per second (0 for unlimited) (default: 0)\n")
	fmt.Print("  -timeout       Request timeout in seconds (default: 10)\n")
	fmt.Print("  -p             Proxy URL (e.g., http://127.0.0.1:8080)\n")
	fmt.Print("  -r             Max retries for failed requests (default: 2)\n")
	fmt.Print("  -f             Follow redirects (true/false) (default: true)\n")	
	fmt.Print("  -h             Show this help message\n\n")
	fmt.Print("Examples:\n")
	fmt.Print("  paraxm -u https://example.com -o results.txt\n")
	fmt.Print("  paraxm -u https://example.com -ratelimit 5 -o results.txt\n")
	fmt.Print("  paraxm -u https://example.com -d 10 -o results.txt\n")
	fmt.Print("  paraxm -u https://example.com -p http://127.0.0.1:8080 -f false\n")
}

func validateFlags(targetPtr *string, outputPtr *string, depthPtr *int,
	threadsPtr *int, timeoutPtr *int, proxyPtr *string,
	retriesPtr *int, followPtr *bool, rateLimitPtr *float64) bool {

	isValid := true

	if *targetPtr == "" {
		pkg.PrintError("Target URL is required\n")
		isValid = false
	} else if !strings.HasPrefix(*targetPtr, "http://") && !strings.HasPrefix(*targetPtr, "https://") {
		pkg.PrintError("Target URL must start with http:// or https://\n")
		isValid = false
	}

	if *outputPtr != "" {

		var dir string
		if filepath.IsAbs(*outputPtr) {
			dir = filepath.Dir(*outputPtr)
		} else if strings.Contains(*outputPtr, "/") {
			dir = filepath.Dir(*outputPtr)
		} else {
			dir = "."
		}
		
		if _, err := os.Stat(dir); os.IsNotExist(err) && dir != "." {
			pkg.PrintError("Output directory %s does not exist\n", dir)
			isValid = false
		}
	}

	if *threadsPtr < 1 || *threadsPtr > 100 {
		pkg.PrintError("Threads must be between 1 and 100\n")
		isValid = false
	}
	
	if *rateLimitPtr < 0 {
		pkg.PrintError("Rate limit must be greater than or equal to 0\n")
		isValid = false
	}

	if *timeoutPtr < 1 || *timeoutPtr > 60 {
		pkg.PrintError("Timeout must be between 1 and 60 seconds\n")
		isValid = false
	}

	if *proxyPtr != "" && !strings.HasPrefix(*proxyPtr, "http://") && !strings.HasPrefix(*proxyPtr, "https://") && !strings.HasPrefix(*proxyPtr, "socks5://") {
		pkg.PrintError("Proxy must be in format http://host:port, https://host:port or socks5://host:port\n")
		isValid = false
	}

	if *retriesPtr < 0 || *retriesPtr > 10 {
		pkg.PrintError("Max retries must be between 0 and 10\n")
		isValid = false
	}

	return isValid
}

func saveResults(outputFile string, results []pkg.Result) error {
	if outputFile == "" {
		return nil
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, result := range results {
		fmt.Fprintf(file, "%s?%s=FUZZ\n", result.URL, result.Parameter)
	}
	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	targetPtr := fs.String("u", "", "Target URL to scan")
	outputPtr := fs.String("o", "", "Output file to save results")
	depthPtr := fs.Int("d", 2, "Crawling depth (1-10)")
	threadsPtr := fs.Int("t", 10, "Number of concurrent threads")
	rateLimitPtr := fs.Float64("ratelimit", 0, "Maximum requests per second")
	timeoutPtr := fs.Int("timeout", 10, "Request timeout in seconds")
	proxyPtr := fs.String("p", "", "Proxy URL (e.g., http://127.0.0.1:8080)")
	retriesPtr := fs.Int("r", 2, "Max retries for failed requests")
	followPtr := fs.Bool("f", true, "Follow redirects (true/false)")
	helpPtr := fs.Bool("h", false, "Show help")

	printBanner()

	err := fs.Parse(os.Args[1:])

	if err != nil {
		if err == flag.ErrHelp {
			printUsage()
			return
		}

		printUsage()
		errMsg := err.Error()
		if strings.Contains(errMsg, "flag needs an argument") {
			parts := strings.Split(errMsg, ":")
			if len(parts) > 1 {
				flagName := strings.TrimSpace(parts[1])
				pkg.PrintError("Flag %s requires a value\n\n", flagName)
			} else {
				pkg.PrintError("%s\n\n", errMsg)
			}
		} else {
			pkg.PrintError("%s\n\n", errMsg)
		}
		return
	}

	if *helpPtr || fs.NFlag() == 0 {
		printUsage()
		return
	}

	if !validateFlags(targetPtr, outputPtr, depthPtr, threadsPtr, timeoutPtr, proxyPtr, retriesPtr, followPtr, rateLimitPtr) {
        return
    }

	config := &pkg.Config{
		Target:          *targetPtr,
		OutputFile:      *outputPtr,
		Depth:           *depthPtr,
		Threads:         *threadsPtr,
		Timeout:         *timeoutPtr,
		Proxy:           *proxyPtr,
		UserAgents:      pkg.GetUserAgents(),
		FollowRedirects: *followPtr,
		MaxRetries:      *retriesPtr,
		RateLimit:       *rateLimitPtr,
		VisitedURLs:     make(map[string]bool),
		VisitedParams:   make(map[string]bool),
		ResultChan:      make(chan pkg.Result, 1000),
		WaitGroup:       sync.WaitGroup{},
	}

	results, err := pkg.RunScan(config)
	if err != nil {
		pkg.PrintError("Scan error: %v\n", err)
		os.Exit(0)
	}

	if len(results) > 0 {
		pkg.PrintInfo("Found %d parameters:\n", len(results))
		for _, result := range results {
			pkg.PrintFound("%s?%s=FUZZ\n", result.URL, result.Parameter)
		}
	} else {
		pkg.PrintInfo("No parameters found.\n")
	}

	if config.OutputFile != "" {
		pkg.PrintInfo("Results saved to: %s\n", config.OutputFile)
	}
}
