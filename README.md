# Wget - File Downloader

A feature-rich command-line file downloader implemented in Go, inspired by GNU Wget. This tool supports single file downloads, batch downloads, rate limiting, background downloads, and website mirroring.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Basic Download](#basic-download)
  - [Custom Output Name](#custom-output-name)
  - [Custom Output Directory](#custom-output-directory)
  - [Rate Limiting](#rate-limiting)
  - [Background Download](#background-download)
  - [Batch Download](#batch-download)
  - [Website Mirroring](#website-mirroring)
- [Examples](#examples)
- [Project Structure](#project-structure)
- [Implementation Details](#implementation-details)

## Features

✅ **Single File Download** - Download files from HTTP/HTTPS URLs  
✅ **Custom Output** - Specify custom file names and directories  
✅ **Rate Limiting** - Control download speed (supports k, M, G units)  
✅ **Background Downloads** - Download files in background with log output  
✅ **Batch Downloads** - Download multiple files asynchronously from a list  
✅ **Website Mirroring** - Download entire websites for offline viewing  
✅ **Progress Bar** - Real-time download progress with speed and ETA  
✅ **File Filtering** - Reject specific file types during mirroring  
✅ **Directory Exclusion** - Exclude specific directories during mirroring  
✅ **Link Conversion** - Convert links for offline browsing  

## Installation

### Prerequisites

- Go 1.24.6 or higher

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd wget

# Build the executable
go build -o wget.exe .

# Or run directly
go run . [OPTIONS] URL
```

## Usage

### Basic Download

Download a file from a URL:

```bash
./wget https://example.com/file.zip
```

**Output:**
```
start at 2025-10-16 01:26:29
sending request, awaiting response... status 200 OK
content size: 20900490 [~19.93MB]
saving file to: ./file.zip
 19.93 MiB / 19.93 MiB [========================================] 100.00% 5.24 MiB/s 0s
Downloaded [https://example.com/file.zip]
finished at 2025-10-16 01:26:33
```

### Custom Output Name

Save the downloaded file with a different name using the `-O` flag:

```bash
./wget -O custom_name.zip https://example.com/file.zip
```

### Custom Output Directory

Save the file to a specific directory using the `-P` flag:

```bash
# Save to a relative directory
./wget -P ./downloads/ https://example.com/file.zip

# Save to home directory
./wget -P ~/Downloads/ https://example.com/file.zip

# Combine with custom name
./wget -O myfile.zip -P ./downloads/ https://example.com/file.zip
```

### Rate Limiting

Limit the download speed using the `--rate-limit` flag:

```bash
# Limit to 300 KB/s
./wget --rate-limit=300k https://example.com/file.zip

# Limit to 2 MB/s
./wget --rate-limit=2M https://example.com/file.zip
```

**Supported units:**
- `k` or `kb` - Kilobytes per second
- `m` or `mb` - Megabytes per second
- `g` or `gb` - Gigabytes per second

### Background Download

Download a file in the background with output redirected to a log file:

```bash
./wget -B https://example.com/file.zip
```

**Output:**
```
Output will be written to "wget-log".
```

Check the log file:
```bash
cat wget-log
```

### Batch Download

Download multiple files asynchronously from a text file:

1. Create a text file with URLs (one per line):

```bash
# downloads.txt
https://example.com/file1.zip
https://example.com/file2.zip
https://example.com/file3.zip
```

2. Run the batch download:

```bash
./wget -i downloads.txt
```

**Features:**
- Downloads all files **asynchronously** (in parallel)
- Shows content sizes before starting
- Reports completion for each file
- Displays final summary

### Website Mirroring

Download an entire website for offline viewing:

```bash
./wget --mirror https://example.com
```

This creates a folder named after the domain (e.g., `example.com`) containing the entire website structure.

#### Mirror with File Type Rejection

Exclude specific file types using `-R` or `--reject`:

```bash
# Exclude images
./wget --mirror -R=jpg,gif,png https://example.com

# Exclude videos
./wget --mirror --reject=mp4,avi https://example.com
```

#### Mirror with Directory Exclusion

Exclude specific directories using `-X` or `--exclude`:

```bash
# Exclude /img directory
./wget --mirror -X=/img https://example.com

# Exclude multiple directories
./wget --mirror -X=/assets,/css https://example.com
```

#### Mirror with Link Conversion

Convert links for offline browsing using `--convert-links`:

```bash
./wget --mirror --convert-links https://example.com
```

This converts absolute URLs to relative paths so the website works offline.

#### Combined Mirror Options

```bash
./wget --mirror -R=jpg,gif -X=/assets --convert-links https://example.com
```

## Examples

### Example 1: Download with Custom Name and Location

```bash
./wget -O report.pdf -P ~/Documents/ https://example.com/annual-report-2024.pdf
```

### Example 2: Rate-Limited Download

```bash
./wget --rate-limit=500k https://example.com/large-file.zip
```

### Example 3: Background Batch Download

```bash
./wget -B -i downloads.txt
```

### Example 4: Mirror Website Without Images

```bash
./wget --mirror -R=jpg,jpeg,png,gif,svg https://example.com
```

### Example 5: Mirror Specific Parts of Website

```bash
./wget --mirror -X=/admin,/private --convert-links https://example.com
```

## Project Structure

```
wget/
├── main.go                          # Entry point and CLI argument parsing
├── go.mod                           # Go module dependencies
├── go.sum                           # Dependency checksums
├── README.md                        # This file
└── internal/
    ├── downloader/
    │   ├── downloader.go           # Core download logic
    │   └── progress.go             # Progress tracking (unused)
    ├── logging/
    │   └── logging.go              # Logging and output formatting
    ├── bg/
    │   └── background.go           # Background download handler
    ├── batch/
    │   └── batch.go                # Batch download from file
    └── mirror/
        ├── mirror.go               # Website mirroring logic
        ├── parser.go               # HTML/CSS resource extraction
        └── convert.go              # Link conversion for offline viewing
```

## Implementation Details

### Core Components

#### 1. Downloader (`internal/downloader/`)

- **HTTP Client**: Uses Go's `net/http` with configurable timeouts
- **Progress Tracking**: Real-time progress bar with download speed and ETA
- **Rate Limiting**: Uses `golang.org/x/time/rate` for precise rate control
- **Path Resolution**: Handles relative paths, home directory expansion (`~`)

**Key Features:**
- Automatic directory creation
- Content-Length detection for progress calculation
- Graceful error handling
- Support for various file types

#### 2. Logging (`internal/logging/`)

- **Formatted Output**: Time-stamped logs with consistent formatting
- **Progress Bar**: Dynamic progress display with:
  - Downloaded/Total size (KiB, MiB, GiB)
  - Percentage complete
  - Download speed
  - Estimated time remaining
- **Background Mode**: Redirects output to `wget-log` file

#### 3. Batch Downloader (`internal/batch/`)

- **Asynchronous Downloads**: Uses goroutines for parallel downloads
- **File Parsing**: Handles various text encodings (UTF-8, UTF-16)
- **BOM Handling**: Removes Byte Order Marks automatically
- **Error Aggregation**: Collects and reports errors for all downloads

#### 4. Website Mirror (`internal/mirror/`)

- **Recursive Crawling**: Follows links up to a configurable depth
- **Resource Extraction**: Parses HTML and CSS for linked resources
- **Domain Filtering**: Only downloads resources from the same domain
- **File System Structure**: Preserves website directory structure
- **Concurrent Downloads**: Processes multiple resources efficiently

**Resource Types Supported:**
- HTML pages (`<a href>`, `<link>`)
- Images (`<img src>`)
- Stylesheets (`<link rel="stylesheet">`)
- JavaScript (`<script src>`)
- CSS imports (`@import`)
- CSS resources (`url()`)

#### 5. Link Converter (`internal/mirror/convert.go`)

- **Relative Path Conversion**: Converts absolute URLs to relative paths
- **Cross-Platform**: Handles both Windows and Unix path separators
- **HTML/CSS Support**: Converts links in both HTML and CSS files
- **Offline Compatibility**: Ensures websites work without internet

### Rate Limiting Algorithm

The rate limiter uses a token bucket algorithm:

1. **Burst Size**: Set to 32KB (minimum read buffer size)
2. **Token Refill**: Tokens refill at the specified rate (bytes/second)
3. **Wait Mechanism**: Blocks read operations until tokens are available
4. **Timeout Handling**: Disables HTTP timeout when rate limiting is active

### Error Handling

- **Network Errors**: Graceful handling of connection failures
- **HTTP Errors**: Reports non-200 status codes
- **File System Errors**: Handles permission and disk space issues
- **Timeout Protection**: Adjusts timeouts based on rate limits

### Performance Optimizations

- **Asynchronous I/O**: Non-blocking downloads for batch operations
- **Buffer Management**: Efficient memory usage with streaming
- **Progress Updates**: Throttled to 100ms intervals to reduce overhead
- **Concurrent Mirroring**: Parallel resource downloads during mirroring

## Dependencies

```go
require (
    golang.org/x/net v0.44.0    // HTML parsing
    golang.org/x/time v0.13.0   // Rate limiting
)
```

## Technical Specifications

- **Language**: Go 1.24.6
- **HTTP Protocol**: HTTP/1.1 and HTTP/2
- **Supported Schemes**: HTTP, HTTPS
- **Default Timeout**: 30 seconds (disabled with rate limiting)
- **Read Buffer**: 32KB
- **Max Mirror Depth**: 5 levels (default)
- **Max Files**: 1000 files per mirror (default)

## Limitations

- Does not support FTP protocol
- Does not support authentication (basic auth, cookies)
- Does not support proxy configuration
- Mirror depth and file limits are hardcoded
- No resume capability for interrupted downloads

## Future Enhancements

- [ ] Resume interrupted downloads
- [ ] FTP protocol support
- [ ] Authentication support
- [ ] Proxy configuration
- [ ] Configurable mirror depth and file limits
- [ ] Checksum verification
- [ ] Compression support
- [ ] IPv6 support

## License

This project is part of the 01-edu curriculum.

## Author

Created as part of the Reboot01 program.
