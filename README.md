# ParaXm - URL Parameter Hunter


<p align="center">
  <img src="logo/ParaXm.jpg" alt="ParaXm Logo" width="200"/>
  <br>
  <br>
  <b>Powerful URL Parameter Discovery Tool for Web App Security Testing</b>
</p>



## Overview

ParaXm is a high-performance URL parameter discovery tool built for security researchers and pentesters who need results. It tears through web applications to find endpoints with parameters that could be vulnerable to XSS, SQLi, SSRF, and plenty of other fun attack vectors.


### Key Features

- **Intelligent Crawling**: Digs deep through HTML, JavaScript, and JSON to uncover hidden endpoints
- **Parameter Detection**: Sniffs out URLs with query parameters across the entire application
- **Concurrent Scanning**: Multi-threaded architecture that gets the job done fast
- **Smart Parsing**: Extracts URLs from everywhere that matters:
  - HTML attributes (href, src, action)
  - JavaScript variables and strings
  - AJAX/Fetch/Axios requests
  - JSON configurations
  - API endpoints you didn't even know existed


## Installation

### Prerequisites

- Go 1.16 or higher

### Building from Source

```bash
# Clone the repository
git clone https://github.com/ZyFoxX/ParaXm.git
cd ParaXm

# Build the project
go build -o ParaXm ParaXm.go
chmod +x ParaXm
```


## Usage

### Basic Usage

Scan a single URL:
```bash
./ParaXm -u https://example.com
```

Scan multiple URLs from a file:
```bash
./ParaXm -l urls.txt
```

Save results to a file:
```bash
./ParaXm -u https://example.com -o results.txt
```

### Advanced Options

```bash
./ParaXm -u https://example.com -t 20 -d 2 -timeout 30
```


### Command-Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `-u` | Single URL to scan | - |
| `-l` | File containing URLs to scan | - |
| `-o` | Output file to write results | - |
| `-t` | Number of concurrent threads | 10 |
| `-timeout` | Timeout in seconds for HTTP requests | 15 |
| `-d` | Delay between requests in seconds | 1 |
| `-h` | Show help | - |


## How It Works

1. **URL Input**: ParaXm takes a URL or a list of URLs as input
2. **Initial Scan**: Hits each URL to get initial content
3. **Content Analysis**: Tears apart the HTML, JavaScript, and other content looking for:
   - Links (via href attributes)
   - Resource URLs (via src attributes)
   - Form submission endpoints (via action attributes)
   - API endpoints in JavaScript
   - AJAX/Fetch request URLs
   - JavaScript string variables containing paths
4. **Parameter Identification**: Flags all URLs with parameters
5. **Recursive Crawling**: Follows discovered URLs within the same domain
6. **Result Collection**: Serves up a nice list of unique parameter-laden URLs


## Detection Capabilities

ParaXm can spot parameters in all these formats:

- Standard query parameters (`?param=value`)
- Hash fragments with parameters (`#/path?param=value`)
- URL path parameters (`/api/v1/{param}/resource`)
- API endpoints with potential parameters
- Form submission endpoints
- AJAX/Fetch request endpoints
- JavaScript-defined endpoints


## Use Cases

- Discovering hidden API endpoints
- Finding potential injection points for security testing
- Mapping web application attack surface
- Identifying parameter-based vulnerabilities


## Disclaimer

ParaXm is designed for legal security testing with proper authorization. The authors are not responsible for any misuse or damage caused by this tool. Use it responsibly!


## License  

**Apache License 2.0** - Free to use, modify, and distribute under the terms of the Apache License. See [LICENSE](LICENSE) for details.
