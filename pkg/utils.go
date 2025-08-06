package pkg

import (
	"fmt"
	"net/url"
	"strings"
	"regexp"
)

var (
	InfoTag  = "[INFO]"
	FoundTag = "[FOUND]"
	ErrorTag = "[ERROR]"
)

func PrintInfo(format string, a ...interface{}) {
	fmt.Print(InfoTag + " ")
	fmt.Printf(format, a...)
}

func PrintFound(format string, a ...interface{}) {
	fmt.Print(FoundTag + " ")
	fmt.Printf(format, a...)
}

func PrintError(format string, a ...interface{}) {
	fmt.Print(ErrorTag + " ")
	fmt.Printf(format, a...)
}

func IsValidParam(param string) bool {
    if len(param) < 2 {
        return false
    }

    valid := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
    if !valid.MatchString(param) {
        return false
    }

    invalidChars := []string{"<", ">", "{", "}", "(", ")", "\\", "\"", "'", ";", "$", "#", "/"}
    for _, char := range invalidChars {
        if strings.Contains(param, char) {
            return false
        }
    }

    jsKeywords := []string{
        "function", "var", "let", "const", "if", "else", "for", "while", "return",
        "break", "case", "catch", "class", "continue", "debugger", "default", "delete",
        "do", "export", "extends", "finally", "import", "in", "instanceof", "new",
        "super", "switch", "this", "throw", "try", "typeof", "void", "with", "yield",
        "null", "true", "false", "undefined",
    }

    for _, kw := range jsKeywords {
        if param == kw {
            return false
        }
    }

    return true
}

func ResolveURL(baseURL, relativeURL string) string {
	if relativeURL == "" {
		return baseURL
	}

	if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
		return relativeURL
	}

	baseURLObj, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	relativeURLObj, err := url.Parse(relativeURL)
	if err != nil {
		return baseURL
	}

	resolvedURL := baseURLObj.ResolveReference(relativeURLObj)
	return resolvedURL.String()
}
