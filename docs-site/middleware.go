package docssite

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DocsContentNegotiationMiddleware handles Accept header-based content negotiation
// for the Hugo documentation site with dual rendering (HTML/Markdown)
func DocsContentNegotiationMiddleware(next http.Handler, docsDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle documentation requests
		if strings.HasPrefix(r.URL.Path, "/docs") || r.URL.Path == "/" {
			// Remove /docs prefix for Hugo site
			path := strings.TrimPrefix(r.URL.Path, "/docs")
			if path == "" {
				path = "/"
			}

			// Handle content negotiation
			accept := r.Header.Get("Accept")

			// Check for markdown request
			if strings.Contains(accept, "text/markdown") {
				serveMarkdownVersion(w, r, docsDir, path)
				return
			}

			// Check for JSON request (for search index)
			if strings.Contains(accept, "application/json") && path == "/search-index.json" {
				http.ServeFile(w, r, filepath.Join(docsDir, "public", "search-index.json"))
				return
			}

			// Check for OpenAPI spec
			if path == "/api/openapi.json" {
				http.ServeFile(w, r, filepath.Join(docsDir, "public", "api", "openapi.json"))
				return
			}

			// Default to Hugo-generated HTML
			serveHugoContent(w, r, docsDir, path)
			return
		}

		// Pass through to next handler
		next.ServeHTTP(w, r)
	})
}

// serveMarkdownVersion serves the markdown version of a page
func serveMarkdownVersion(w http.ResponseWriter, r *http.Request, docsDir, path string) {
	// Try to find markdown file
	mdPath := filepath.Join(docsDir, "content")

	// Handle root path
	if path == "/" || path == "" {
		mdPath = filepath.Join(mdPath, "_index.md")
	} else {
		// Remove trailing slash
		path = strings.TrimSuffix(path, "/")

		// Try different markdown file locations
		possiblePaths := []string{
			filepath.Join(mdPath, path+".md"),
			filepath.Join(mdPath, path, "_index.md"),
			filepath.Join(mdPath, path, "index.md"),
		}

		found := false
		for _, p := range possiblePaths {
			if _, err := os.Stat(p); err == nil {
				mdPath = p
				found = true
				break
			}
		}

		if !found {
			http.NotFound(w, r)
			return
		}
	}

	// Read and serve markdown file
	content, err := os.ReadFile(mdPath)
	if err != nil {
		http.Error(w, "Error reading markdown file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	if _, err := w.Write(content); err != nil {
		slog.Error("write markdown response", "error", err)
	}
}

// serveHugoContent serves Hugo-generated HTML content
func serveHugoContent(w http.ResponseWriter, r *http.Request, docsDir, path string) {
	// Build the file path for Hugo-generated content
	filePath := filepath.Join(docsDir, "public")

	// Handle root path
	if path == "/" || path == "" {
		filePath = filepath.Join(filePath, "index.html")
	} else {
		// Check if it's a directory (needs index.html)
		fullPath := filepath.Join(filePath, path)
		if stat, err := os.Stat(fullPath); err == nil && stat.IsDir() {
			filePath = filepath.Join(fullPath, "index.html")
		} else {
			// Check for .html extension
			if !strings.HasSuffix(path, ".html") {
				filePath = filepath.Join(filePath, path+".html")
			} else {
				filePath = fullPath
			}
		}
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Try with index.html
		if !strings.HasSuffix(filePath, "index.html") {
			filePath = filepath.Join(filepath.Dir(filePath), "index.html")
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}

// GenerateSearchIndex creates a search index JSON file from Hugo content
func GenerateSearchIndex(docsDir string) error {
	// This would parse all Hugo content files and create a search index
	// For now, we'll create a simple placeholder
	searchIndex := []map[string]interface{}{
		{
			"url":         "/docs/",
			"title":       "Dark Pawns Documentation",
			"description": "Documentation for Dark Pawns MUD - A resurrection of the 1997-2010 MUD with AI agents as first-class players",
			"content":     "Welcome to Dark Pawns documentation. This site provides comprehensive documentation for the Dark Pawns MUD resurrection project.",
			"tags":        []string{"documentation", "home"},
		},
		{
			"url":         "/docs/getting-started/",
			"title":       "Getting Started",
			"description": "Quick start guide for Dark Pawns",
			"content":     "Learn how to get started with Dark Pawns, whether you're a player or an agent developer.",
			"tags":        []string{"getting-started", "guide"},
		},
		{
			"url":         "/docs/api/",
			"title":       "API Reference",
			"description": "Complete API documentation for Dark Pawns",
			"content":     "WebSocket and REST API documentation for integrating with Dark Pawns.",
			"tags":        []string{"api", "reference", "websocket"},
		},
	}

	// Create JSON file
	jsonContent := "["
	for i, item := range searchIndex {
		if i > 0 {
			jsonContent += ","
		}
		jsonContent += fmt.Sprintf(`{"url":"%s","title":"%s","description":"%s","content":"%s","tags":["%s"]}`,
			item["url"], item["title"], item["description"], item["content"],
			strings.Join(item["tags"].([]string), `","`))
	}
	jsonContent += "]"

	// Write to file
	indexPath := filepath.Join(docsDir, "public", "search-index.json")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		return fmt.Errorf("create search index directory: %w", err)
	}
	return os.WriteFile(indexPath, []byte(jsonContent), 0644)
}
