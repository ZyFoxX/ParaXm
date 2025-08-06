package pkg

import (
	"fmt"
	"io"
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/time/rate"
)

type Config struct {
	Target          string
	OutputFile      string
	Depth           int
	Threads         int
	Timeout         int
	Proxy           string
	UserAgents      []string
	FollowRedirects bool
	MaxRetries      int
	RateLimit       float64
	RateLimiter     *rate.Limiter
	VisitedURLs     map[string]bool
	VisitedParams   map[string]bool
	VisitedMutex    sync.RWMutex
	ResultChan      chan Result
	WaitGroup       sync.WaitGroup
	Client          *http.Client
}

type Result struct {
	URL         string
	Parameter   string
	Source      string
	Method      string
	ContentType string
	StatusCode  int
	Timestamp   time.Time
}

func RunScan(config *Config) ([]Result, error) {
	config.Client = InitClient(config)
	
	if config.RateLimit > 0 {
		config.RateLimiter = rate.NewLimiter(rate.Limit(config.RateLimit), config.Threads)
	}

	PrintInfo("Target: %s\n", config.Target)
	PrintInfo("Starting scan...\n")

	var results []Result
	var resultsMutex sync.Mutex

	go func() {
		for result := range config.ResultChan {
			resultsMutex.Lock()
			results = append(results, result)
			resultsMutex.Unlock()
		}
	}()

	config.WaitGroup.Add(1)
	go func() {
		defer config.WaitGroup.Done()
		crawlURL(config, config.Target, 1)
	}()

	config.WaitGroup.Wait()
	close(config.ResultChan)

	return results, nil
}

func crawlURL(config *Config, targetURL string, depth int) {
	if depth > config.Depth {
		return
	}

	config.VisitedMutex.RLock()
	visited := config.VisitedURLs[targetURL]
	config.VisitedMutex.RUnlock()

	if visited {
		return
	}

	config.VisitedMutex.Lock()
	config.VisitedURLs[targetURL] = true
	config.VisitedMutex.Unlock()


	if config.RateLimiter != nil {	
		err := config.RateLimiter.Wait(context.Background())
		if err != nil {
			return
		}
	}

	method := "GET"

	var resp *http.Response
	var err error
	var body []byte

	for retry := 0; retry <= config.MaxRetries; retry++ {
		req, err := http.NewRequest(method, targetURL, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", GetRandomUserAgent())

		resp, err = config.Client.Do(req)

		if err != nil {
			if retry < config.MaxRetries {
				time.Sleep(time.Duration(retry+1) * time.Second)
				continue
			}
			return
		}

		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			if retry < config.MaxRetries {
				continue
			}
			return
		}

		break
	}

	if resp == nil || err != nil {
		return
	}

	content := string(body)
	contentType := resp.Header.Get("Content-Type")

	extractParameters(config, targetURL, content, method, contentType, resp.StatusCode)

	if strings.Contains(contentType, "text/html") {
		urls := extractURLs(config, targetURL, content)
		for _, u := range urls {
			config.WaitGroup.Add(1)
			go func(url string) {
				defer config.WaitGroup.Done()
				crawlURL(config, url, depth+1)
			}(u)
		}
	}
}

func extractParameters(config *Config, baseURL, content, method, contentType string, statusCode int) {
	if strings.Contains(contentType, "text/html") {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err == nil {
			doc.Find("form").Each(func(i int, form *goquery.Selection) {
				formAction, _ := form.Attr("action")
				formMethod, _ := form.Attr("method")
				if formMethod == "" {
					formMethod = "GET"
				}
				formMethod = strings.ToUpper(formMethod)

				formActionURL := ResolveURL(baseURL, formAction)

				form.Find("input, select, textarea").Each(func(j int, input *goquery.Selection) {
					paramName, exists := input.Attr("name")
					if exists && IsValidParam(paramName) {
						addResult(config, formActionURL, paramName, "HTML-Form", formMethod, contentType, statusCode)
					}
				})
			})

			doc.Find("a[href]").Each(func(i int, link *goquery.Selection) {
				href, _ := link.Attr("href")
				processURLForParams(config, baseURL, href, "HTML-Link", method, contentType, statusCode)
			})

			urlAttributes := []string{"src", "data", "action", "formaction", "ping"}
			for _, attr := range urlAttributes {
				selector := fmt.Sprintf("[%s]", attr)
				doc.Find(selector).Each(func(i int, el *goquery.Selection) {
					attrVal, _ := el.Attr(attr)
					processURLForParams(config, baseURL, attrVal, "HTML-Attr", method, contentType, statusCode)
				})
			}
		}
	}

	jsPatterns := []string{
        `(?:fetch|axios|ajax|XMLHttpRequest|xhr)\s*\(\s*["']([^"'?#]+)\?([^"'#]+)`,
        `\.(?:get|post|put|delete|patch|request)\s*\(\s*["']([^"'?#]+)\?([^"'#]+)`,
        `new\s+XMLHttpRequest\s*\([^)]*\)[^{]*\.open\s*\(\s*["'][^"']*["']\s*,\s*["']([^"'?#]+)\?([^"'#]+)`,
        `\.send\s*\(\s*([^)]+)`,
        `new\s+URL\s*\(\s*["']([^"'?#]+)\?([^"'#]+)`,
        `(?:URL|url)\.searchParams\.[a-zA-Z]+\s*\(\s*["']([^"']+)["']`,
        `(?:location|window\.location|document\.location)(?:\.href|\["href"\]|\.assign|\.replace)\s*=\s*["']([^"'?#]+)\?([^"'#]+)`,
        `(?:url|href|src|data-url|data-src)\s*[:=]\s*["']([^"'?#]+)\?([^"'#]+)`,
        `(?:params|parameters|query|data|body)\s*:\s*({[^}]+}|\[[^\]]+\]|"[^"]+"|'[^']+'|\w+)`,
        `(?:params|parameters|query|data|body)\s*=\s*({[^}]+}|\[[^\]]+\]|"[^"]+"|'[^']+'|\w+)`,
        "`([^`?#]+)\\?([^`#]+)",
        `(?:FormData|URLSearchParams)\s*\(\s*([^)]+)`,
        `\.append\s*\(\s*["']([^"']+)["']\s*,\s*["']?([^"')]+)`,
        `\$\s*\.(?:get|post|ajax)\s*\(\s*{[^}]*url\s*:\s*["']([^"'?#]+)\?([^"'#]+)`,
    }

	for _, pattern := range jsPatterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) >= 3 {
				baseURLPart := match[1]
				queryPart := match[2]

				queryParams := strings.Split(queryPart, "&")
				for _, param := range queryParams {
					if strings.Contains(param, "=") {
						paramParts := strings.SplitN(param, "=", 2)
						paramName := paramParts[0]

						if IsValidParam(paramName) {
							absoluteURL := ResolveURL(baseURL, baseURLPart)
							addResult(config, absoluteURL, paramName, "JavaScript", method, contentType, statusCode)
						}
					}
				}
			} else if len(match) >= 2 {
				objContent := match[1]
				keyRegex := regexp.MustCompile(`["']?([a-zA-Z0-9_]+)["']?\s*:`)
				keyMatches := keyRegex.FindAllStringSubmatch(objContent, -1)

				for _, keyMatch := range keyMatches {
					if len(keyMatch) >= 2 {
						paramName := keyMatch[1]

						if IsValidParam(paramName) {
							addResult(config, baseURL, paramName, "JavaScript-Object", method, contentType, statusCode)
						}
					}
				}
			}
		}
	}

	urlRegex := regexp.MustCompile(`https?://[^\s'"<>()]+`)
	matches := urlRegex.FindAllString(content, -1)

	for _, match := range matches {
		processURLForParams(config, baseURL, match, "URL-In-Content", method, contentType, statusCode)
	}

	commentRegex := regexp.MustCompile(`<!--([\s\S]*?)-->|//([^\n]*)|/\*([\s\S]*?)\*/`)
	commentMatches := commentRegex.FindAllStringSubmatch(content, -1)

	for _, comment := range commentMatches {
		var commentContent string
		for i := 1; i < len(comment); i++ {
			if comment[i] != "" {
				commentContent = comment[i]
				break
			}
		}

		paramRegex := regexp.MustCompile(`[?&]([a-zA-Z0-9_-]+)=`)
		paramMatches := paramRegex.FindAllStringSubmatch(commentContent, -1)

		for _, paramMatch := range paramMatches {
			if len(paramMatch) >= 2 {
				paramName := paramMatch[1]

				if IsValidParam(paramName) {
					addResult(config, baseURL, paramName, "Comment", method, contentType, statusCode)
				}
			}
		}

		urlMatches := urlRegex.FindAllString(commentContent, -1)
		for _, urlMatch := range urlMatches {
			processURLForParams(config, baseURL, urlMatch, "URL-In-Comment", method, contentType, statusCode)
		}
	}
}

func addResult(config *Config, url, param, source, method, contentType string, statusCode int) {
	paramKey := url + "|" + param

	config.VisitedMutex.RLock()
	visited := config.VisitedParams[paramKey]
	config.VisitedMutex.RUnlock()

	if !visited {
		config.VisitedMutex.Lock()
		config.VisitedParams[paramKey] = true
		config.VisitedMutex.Unlock()

		config.ResultChan <- Result{
			URL:         url,
			Parameter:   param,
			Source:      source,
			Method:      method,
			ContentType: contentType,
			StatusCode:  statusCode,
			Timestamp:   time.Now(),
		}
	}
}

func processURLForParams(config *Config, baseURL, urlStr, source, method, contentType string, statusCode int) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return
	}

	if !parsedURL.IsAbs() {
		baseURLObj, err := url.Parse(baseURL)
		if err != nil {
			return
		}

		parsedURL = baseURLObj.ResolveReference(parsedURL)
	}

	query := parsedURL.Query()
	for paramName := range query {
		if IsValidParam(paramName) {
			absoluteURL := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
			addResult(config, absoluteURL, paramName, source, method, contentType, statusCode)
		}
	}
}

func extractURLs(config *Config, baseURL string, content string) []string {
	var urls []string
	var uniqueURLs = make(map[string]bool)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return urls
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return urls
	}
	baseHost := parsedBase.Host

	selectors := map[string]string{
    	"a":          "href",
    	"script":     "src",
    	"link":       "href",
    	"img":        "src",
    	"form":       "action",
    	"iframe":     "src",
    	"frame":      "src",
    	"embed":      "src",
    	"object":     "data",
    	"video":      "src",
    	"audio":      "src",
    	"source":     "src",
    	"track":      "src",
    	"area":       "href",
    	"base":       "href",
    	"portal":     "src",
    	"picture":    "srcset",
    	"img[srcset]": "srcset",
    	"[data-ajax-url]":    "data-ajax-url",
    	"[data-url]":         "data-url",
    	"[data-src]":         "data-src",
    	"[data-href]":        "data-href",
    	"meta[property='og:url']": "content",
    	"meta[name='twitter:url']": "content",
    	"meta[itemprop='url']":    "content",
    	"use":       "xlink:href|href",
    	"image":     "xlink:href|href",
    	"[ng-href]":  "ng-href",
    	"[x-route]":  "x-route",
    }

	for selector, attr := range selectors {
		doc.Find(selector).Each(func(i int, el *goquery.Selection) {
			attrVal, exists := el.Attr(attr)
			if exists {
				absoluteURL := ResolveURL(baseURL, attrVal)

				if !strings.HasPrefix(absoluteURL, "http://") && !strings.HasPrefix(absoluteURL, "https://") {
					return
				}

				if uniqueURLs[absoluteURL] {
					return
				}

				parsedURL, err := url.Parse(absoluteURL)
				if err != nil {
					return
				}

				if parsedURL.Host == baseHost {
					uniqueURLs[absoluteURL] = true
					urls = append(urls, absoluteURL)
				}
			}
		})
	}

	return urls
}
