package scanner

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ZyFoxX/ParaXm/pkg/utils"
)

var (
	urlParamRegex     = regexp.MustCompile(`[\?&]([^=&#]+)=([^&#]*)`)
	endpointRegex     = regexp.MustCompile(`(?:"|')(/[a-zA-Z0-9_?&=./\\-]+)(?:"|')`)
	hrefRegex         = regexp.MustCompile(`href=["']([^"']+)["']`)
	srcRegex          = regexp.MustCompile(`src=["']([^"']+)["']`)
	ajaxRegex         = regexp.MustCompile(`(?i)(?:xhr|fetch|axios|ajax)(?:.|\n)*?(?:['"])([^'"]+)(?:['"])`)
	jsonEndpointRegex = regexp.MustCompile(`["'](?:url|endpoint|api|path|uri|href)["']\s*:\s*["']([^"']+)["']`)
	formActionRegex   = regexp.MustCompile(`<form[^>]*action=["']([^"']+)["']`)
	jsonParamsRegex   = regexp.MustCompile(`["']([^"']+)["']\s*:\s*["']([^"']*)["']`)
)

func ScanURLs(initialURLs []string, numThreads, timeoutSec, delaySec int) []string {
	var wg sync.WaitGroup
	resultChan := make(chan string, 100000)
	urlsToCrawl := make(chan string, 100000)
	semaphore := make(chan struct{}, numThreads)
	
	client := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			MaxIdleConnsPerHost: 100,
		},
	}

	processedURLs := make(map[string]bool)
	urlsMutex := &sync.Mutex{}
	
	delayDuration := time.Duration(delaySec) * time.Second
	delayChan := make(chan struct{}, numThreads)
	
	for i := 0; i < numThreads; i++ {
		delayChan <- struct{}{}
	}
	
	go func() {
		for {
			time.Sleep(delayDuration)
			select {
			case delayChan <- struct{}{}:
			default:
			}
		}
	}()
	
	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for urlStr := range urlsToCrawl {
				<-delayChan
				
				semaphore <- struct{}{}
				foundURLs := scanURL(urlStr, client, resultChan)
				<-semaphore
				
				for _, newURL := range foundURLs {
					normalizedURL := utils.NormalizeURL(newURL)
					
					urlsMutex.Lock()
					processed := processedURLs[normalizedURL]
					if !processed {
						processedURLs[normalizedURL] = true
						urlsMutex.Unlock()
						urlsToCrawl <- normalizedURL
					} else {
						urlsMutex.Unlock()
					}
				}
			}
		}()
	}
	
	go func() {
		for _, u := range initialURLs {
			normalizedURL := utils.NormalizeURL(u)
			
			urlsMutex.Lock()
			if !processedURLs[normalizedURL] {
				processedURLs[normalizedURL] = true
				urlsMutex.Unlock()
				urlsToCrawl <- normalizedURL
			} else {
				urlsMutex.Unlock()
			}
		}
		
		for {
			if len(urlsToCrawl) == 0 {
				time.Sleep(2 * time.Second)
				if len(urlsToCrawl) == 0 {
					break
				}
			}
			
			time.Sleep(500 * time.Millisecond)
		}
		
		close(urlsToCrawl)
	}()
	
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	uniqueResults := make(map[string]bool)
	var results []string
	
	for url := range resultChan {
		if !uniqueResults[url] {
			uniqueResults[url] = true
			results = append(results, url)
		}
	}

	return results
}

func scanURL(urlStr string, client *http.Client, resultChan chan<- string) []string {
	crawlURLs := []string{}

	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	baseURL, err := url.Parse(urlStr)
	if err != nil {
		return crawlURLs
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return crawlURLs
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return crawlURLs
	}
	defer resp.Body.Close()

	if strings.Contains(urlStr, "?") || strings.Contains(urlStr, "&") {
		resultChan <- urlStr
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return crawlURLs
	}
	
	bodyStr := string(body)
	
	hrefMatches := hrefRegex.FindAllStringSubmatch(bodyStr, -1)
	for _, match := range hrefMatches {
		if len(match) > 1 {
			href := match[1]
			
			foundURL := utils.ResolveURL(urlStr, href)
			if foundURL != "" {
				if strings.Contains(foundURL, "?") {
					resultChan <- foundURL
				}
				
				foundURLParsed, err := url.Parse(foundURL)
				if err == nil && foundURLParsed.Hostname() == baseURL.Hostname() {
					crawlURLs = append(crawlURLs, foundURL)
				}
			}
		}
	}
	
	srcMatches := srcRegex.FindAllStringSubmatch(bodyStr, -1)
	for _, match := range srcMatches {
		if len(match) > 1 {
			src := match[1]
			
			foundURL := utils.ResolveURL(urlStr, src)
			if foundURL != "" && strings.Contains(foundURL, "?") {
				resultChan <- foundURL
			}
		}
	}
	
	formMatches := formActionRegex.FindAllStringSubmatch(bodyStr, -1)
	for _, match := range formMatches {
		if len(match) > 1 {
			action := match[1]
			
			foundURL := utils.ResolveURL(urlStr, action)
			if foundURL != "" {
				if strings.Contains(foundURL, "?") {
					resultChan <- foundURL
				}
				
				foundURLParsed, err := url.Parse(foundURL)
				if err == nil && foundURLParsed.Hostname() == baseURL.Hostname() {
					crawlURLs = append(crawlURLs, foundURL)
				}
			}
		}
	}
	
	jsEndpoints := make(map[string]bool)
	
	endpointMatches := endpointRegex.FindAllStringSubmatch(bodyStr, -1)
	for _, match := range endpointMatches {
		if len(match) > 1 {
			endpoint := match[1]
			jsEndpoints[endpoint] = true
		}
	}
	
	ajaxMatches := ajaxRegex.FindAllStringSubmatch(bodyStr, -1)
	for _, match := range ajaxMatches {
		if len(match) > 1 {
			endpoint := match[1]
			jsEndpoints[endpoint] = true
		}
	}
	
	jsonMatches := jsonEndpointRegex.FindAllStringSubmatch(bodyStr, -1)
	for _, match := range jsonMatches {
		if len(match) > 1 {
			endpoint := match[1]
			jsEndpoints[endpoint] = true
		}
	}
	
	for endpoint := range jsEndpoints {
		if strings.Contains(endpoint, "?") || 
		   strings.Contains(endpoint, "/api/") || 
		   strings.Contains(endpoint, "/ajax/") ||
		   strings.Contains(endpoint, "/v1/") ||
		   strings.Contains(endpoint, "/v2/") ||
		   strings.Contains(endpoint, "/rest/") {
			
			foundURL := utils.ResolveURL(urlStr, endpoint)
			if foundURL != "" {
				if strings.Contains(foundURL, "?") {
					resultChan <- foundURL
				}
				
				foundURLParsed, err := url.Parse(foundURL)
				if err == nil && foundURLParsed.Hostname() == baseURL.Hostname() {
					crawlURLs = append(crawlURLs, foundURL)
				}
			}
		}
	}
	
	scriptTags := regexp.MustCompile(`<script[^>]*>([\s\S]*?)</script>`).FindAllStringSubmatch(bodyStr, -1)
	for _, scriptMatch := range scriptTags {
		if len(scriptMatch) > 1 {
			scriptContent := scriptMatch[1]
			
			urlPatterns := regexp.MustCompile(`['"]([^'"]*\?[^'"]+)['"]`).FindAllStringSubmatch(scriptContent, -1)
			for _, urlMatch := range urlPatterns {
				if len(urlMatch) > 1 {
					potentialURL := urlMatch[1]
					if strings.Contains(potentialURL, "=") {
						foundURL := utils.ResolveURL(urlStr, potentialURL)
						if foundURL != "" {
							resultChan <- foundURL
						}
					}
				}
			}
			
			jsonParams := jsonParamsRegex.FindAllStringSubmatch(scriptContent, -1)
			for _, paramMatch := range jsonParams {
				if len(paramMatch) > 2 {
					paramName := paramMatch[1]
					paramValue := paramMatch[2]
					
					if (strings.Contains(paramName, "url") || 
						strings.Contains(paramName, "endpoint") || 
						strings.Contains(paramName, "uri") || 
						strings.Contains(paramName, "link") || 
						strings.Contains(paramName, "href")) && 
						(strings.HasPrefix(paramValue, "/") || 
						strings.HasPrefix(paramValue, "http")) {
						
						foundURL := utils.ResolveURL(urlStr, paramValue)
						if foundURL != "" {
							if strings.Contains(foundURL, "?") {
								resultChan <- foundURL
							}
							
							foundURLParsed, err := url.Parse(foundURL)
							if err == nil && foundURLParsed.Hostname() == baseURL.Hostname() {
								crawlURLs = append(crawlURLs, foundURL)
							}
						}
					}
				}
			}
		}
	}
	
	apiPatterns := []*regexp.Regexp{
		regexp.MustCompile(`api\.([a-zA-Z0-9_-]+)\.([a-zA-Z0-9_-]+)`),
		regexp.MustCompile(`/api/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)`),
		regexp.MustCompile(`/v[0-9]+/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)`),
		regexp.MustCompile(`/ajax/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)`),
	}
	
	for _, pattern := range apiPatterns {
		matches := pattern.FindAllStringSubmatch(bodyStr, -1)
		for _, match := range matches {
			if len(match) > 2 {
				var endpoint string
				if strings.HasPrefix(match[0], "/") {
					endpoint = match[0]
				} else if strings.HasPrefix(match[0], "api.") {
					endpoint = "/api/" + match[1] + "/" + match[2]
				} else {
					continue
				}
				
				foundURL := utils.ResolveURL(urlStr, endpoint)
				if foundURL != "" {
					foundURLParsed, err := url.Parse(foundURL)
					if err == nil && foundURLParsed.Hostname() == baseURL.Hostname() {
						crawlURLs = append(crawlURLs, foundURL)
					}
				}
			}
		}
	}
	
	commonParams := []string{
		"id", "user_id", "post_id", "page", "category", "search", "query", "file", 
		"action", "type", "view", "ref", "token", "lang", "redirect",
	}
	
	for _, param := range commonParams {
		u, err := url.Parse(urlStr)
		if err != nil {
			continue
		}
		
		if u.RawQuery != "" {
			continue
		}
		
		q := u.Query()
		q.Set(param, "1")
		u.RawQuery = q.Encode()
		
		foundURL := u.String()
		foundURLParsed, err := url.Parse(foundURL)
		if err == nil && foundURLParsed.Hostname() == baseURL.Hostname() {
			crawlURLs = append(crawlURLs, foundURL)
		}
	}

	return crawlURLs
}