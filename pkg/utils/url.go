package utils

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
)

func ReadLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

func NormalizeURL(urlStr string) string {
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}
	
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	
	path := parsedURL.Path
	if path == "" {
		path = "/"
	}
	
	parsedURL.Fragment = ""
	
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	parsedURL.Path = path
	
	return parsedURL.String()
}

func ResolveURL(baseURLStr, relativeURL string) string {
	if strings.HasPrefix(relativeURL, "javascript:") || 
	   strings.HasPrefix(relativeURL, "mailto:") || 
	   strings.HasPrefix(relativeURL, "tel:") || 
	   strings.HasPrefix(relativeURL, "data:") ||
	   relativeURL == "#" || 
	   relativeURL == "" {
		return ""
	}
	
	if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
		return relativeURL
	}
	
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return ""
	}
	
	relURL, err := url.Parse(relativeURL)
	if err != nil {
		return ""
	}
	
	resolvedURL := baseURL.ResolveReference(relURL)
	return resolvedURL.String()
}

func SaveResults(results []string, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	
	for _, url := range results {
		writer.WriteString(url + "\n")
	}
	
	writer.Flush()
}