package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// fetchHTML downloads the HTML content from the given URL and returns the root HTML node.
func fetchHTML(url string) *html.Node {
	// Perform HTTP GET request
	resp, err := http.Get(url)
	// Check for errors
	if err != nil {
		return nil
	}
	// Check for non-200 status code
	if resp.StatusCode != http.StatusOK {
		// Log the error
		log.Printf("Failed to fetch %s: %s\n", url, resp.Status)
		// Close the response body
		resp.Body.Close()
		// Return nil if the request was not successful
		return nil
	}
	// Close the response body when done
	defer resp.Body.Close()
	// Parse the HTML content
	node, err := html.Parse(resp.Body)
	// Check for parsing errors
	if err != nil {
		// Return nil if parsing fails
		log.Printf("Failed to parse HTML: %v\n", err)
		// Return nil if the parsing was not successful
		return nil
	}
	// Return the root node of the parsed HTML
	return node
}

// extractLinks walks the HTML node tree and collects href values from <a> tags.
func extractLinks(n *html.Node) []string {
	var links []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					links = append(links, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return links
}

// filterFiles removes directory links (starting or ending with '/') and returns only file names.
func filterFiles(links []string) []string {
	var files []string
	for _, link := range links {
		if strings.HasPrefix(link, "/") || strings.HasSuffix(link, "/") {
			continue
		}
		files = append(files, link)
	}
	return files
}

func downloadFile(baseURL, fileName, outDir string) error {
	// Define allowed extensions inside the function
	allowedExts := []string{".asc", ".asc-ma1", ".asc-pierov", ".apk", ".bspatch", ".dmg", ".exe", ".gz", ".idsig", ".mar", ".txt", ".zip", ".xz"}

	// Inline extension check
	ext := strings.ToLower(filepath.Ext(fileName))
	allowed := false
	for _, e := range allowedExts {
		if ext == e {
			allowed = true
			break
		}
	}
	if !allowed {
		log.Printf("Skipping %s (disallowed extension %s)\n", fileName, ext)
		return nil
	}

	// Check if the file already exists
	outPath := filepath.Join(outDir, fileName)
	if fileExists(outPath) {
		log.Printf("File %s already exists, skipping download.\n", outPath)
		return nil
	}

	// Download the file
	// Construct the full URL
	url := baseURL + fileName
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %v", url, err)
	}
	defer resp.Body.Close()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", outDir, err)
	}

	// Create local file
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", outPath, err)
	}
	defer outFile.Close()

	// Copy response body to file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("error saving %s: %v", outPath, err)
	}

	fmt.Printf("Downloaded %s\n", fileName)
	return nil
}

/*
It checks if the file exists
If the file exists, it returns true
If the file does not exist, it returns false
*/
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

var (
	// Command-line flags
	torVersion = "14.5.1" // Default Tor Browser version
)

func init() {
	// Command-line flags
	version := flag.String("version", "14.5.1", "Tor Browser version to download")
	// Parse command-line flags
	flag.Parse()
	// Check if version is provided
	torVersion = *version
	// Create output directory
	err := os.MkdirAll(torVersion, 0755)
	// Check if directory creation was successful
	if err != nil {
		log.Fatalln("Failed to create output directory:", err)
	}
}

func main() {
	baseURL := fmt.Sprintf("https://dist.torproject.org/torbrowser/%s/", torVersion)

	// Fetch and parse HTML
	node := fetchHTML(baseURL)

	// Extract and filter links
	links := extractLinks(node)
	files := filterFiles(links)

	// Download each file
	for _, file := range files {
		err := downloadFile(baseURL, file, torVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}

	fmt.Printf("All files for version %s have been downloaded into %s/\n", torVersion, torVersion)
}
