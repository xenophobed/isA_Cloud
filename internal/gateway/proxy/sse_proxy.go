package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isa-cloud/isa_cloud/pkg/logger"
)

// SSEProxy handles Server-Sent Events proxying
type SSEProxy struct {
	targetURL string
	logger    *logger.Logger
	timeout   time.Duration
}

// NewSSEProxy creates a new SSE proxy handler
func NewSSEProxy(targetURL string, logger *logger.Logger) *SSEProxy {
	return &SSEProxy{
		targetURL: targetURL,
		logger:    logger,
		timeout:   30 * time.Minute, // Long timeout for SSE connections
	}
}

// Handler returns a Gin handler for SSE proxying
func (p *SSEProxy) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if client accepts SSE
		accept := c.GetHeader("Accept")
		if !strings.Contains(accept, "text/event-stream") && !strings.Contains(accept, "*/*") {
			// Not an SSE request, fall back to normal proxy
			p.proxyHTTP(c)
			return
		}

		// Handle SSE request
		p.proxySSE(c)
	}
}

// proxySSE handles SSE-specific proxying
func (p *SSEProxy) proxySSE(c *gin.Context) {
	// For agents and models services, preserve the full path as they expect /api/v1/{service}/ prefix
	path := c.Request.URL.Path
	
	// Check if this is an agents or models service that needs full path preservation
	if strings.HasPrefix(path, "/api/v1/agents/") || strings.HasPrefix(path, "/api/v1/models/") {
		// Keep the full path including /api/v1/{service}/
		// These services handle their own routing with the full prefix
	} else if strings.HasPrefix(path, "/api/v1/") {
		// For other services, strip the /api/v1/{service} prefix
		parts := strings.SplitN(strings.TrimPrefix(path, "/api/v1/"), "/", 2)
		if len(parts) > 1 {
			// Keep everything after the service name
			path = "/" + parts[1]
		} else {
			// Just the service name, no additional path
			path = "/"
		}
	}
	
	// Build target URL
	targetURL := p.targetURL + path
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	p.logger.Debug("Proxying SSE request", 
		"target", targetURL,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)

	// Create request to target
	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		p.logger.Error("Failed to create SSE proxy request", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create proxy request"})
		return
	}

	// Copy headers
	for key, values := range c.Request.Header {
		// Skip hop-by-hop headers
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Ensure SSE headers
	// MCP requires both application/json and text/event-stream
	if !strings.Contains(req.Header.Get("Accept"), "text/event-stream") {
		currentAccept := req.Header.Get("Accept")
		if currentAccept == "" {
			req.Header.Set("Accept", "application/json, text/event-stream")
		} else {
			req.Header.Set("Accept", currentAccept+", text/event-stream")
		}
	}
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Create HTTP client with longer timeout for SSE
	client := &http.Client{
		Timeout: p.timeout,
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		p.logger.Error("Failed to proxy SSE request", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach target service"})
		return
	}
	defer resp.Body.Close()

	// Check if response is SSE
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		// Not SSE response, copy normally
		p.copyResponse(c, resp)
		return
	}

	// Set SSE headers for client
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering
	
	// Set the status code from upstream response
	c.Status(resp.StatusCode)

	// Stream SSE response
	c.Stream(func(w io.Writer) bool {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			
			// Write line with newline
			if _, err := fmt.Fprintf(w, "%s\n", line); err != nil {
				p.logger.Error("Failed to write SSE data", "error", err)
				return false
			}
			
			// Flush after each event (double newline)
			if line == "" {
				c.Writer.Flush()
			}
		}
		
		if err := scanner.Err(); err != nil && err != io.EOF {
			p.logger.Error("SSE scanner error", "error", err)
		}
		
		return false // End stream
	})
}

// proxyHTTP handles regular HTTP proxying
func (p *SSEProxy) proxyHTTP(c *gin.Context) {
	// For agents and models services, preserve the full path as they expect /api/v1/{service}/ prefix
	path := c.Request.URL.Path
	
	// Check if this is an agents or models service that needs full path preservation
	if strings.HasPrefix(path, "/api/v1/agents/") || strings.HasPrefix(path, "/api/v1/models/") {
		// Keep the full path including /api/v1/{service}/
		// These services handle their own routing with the full prefix
	} else if strings.HasPrefix(path, "/api/v1/") {
		// For other services, strip the /api/v1/{service} prefix
		parts := strings.SplitN(strings.TrimPrefix(path, "/api/v1/"), "/", 2)
		if len(parts) > 1 {
			// Keep everything after the service name
			path = "/" + parts[1]
		} else {
			// Just the service name, no additional path
			path = "/"
		}
	}
	
	// Build target URL
	targetURL := p.targetURL + path
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	p.logger.Debug("Proxying HTTP request",
		"target", targetURL,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)

	// Create request
	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		p.logger.Error("Failed to create proxy request", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create proxy request"})
		return
	}

	// Copy headers
	for key, values := range c.Request.Header {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		p.logger.Error("Failed to proxy request", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach target service"})
		return
	}
	defer resp.Body.Close()

	// Copy response
	p.copyResponse(c, resp)
}

// copyResponse copies HTTP response to client
func (p *SSEProxy) copyResponse(c *gin.Context, resp *http.Response) {
	// Copy headers
	for key, values := range resp.Header {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Set status code
	c.Status(resp.StatusCode)

	// Copy body
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		p.logger.Error("Failed to copy response body", "error", err)
	}
}

// isHopByHopHeader checks if a header is hop-by-hop
func isHopByHopHeader(header string) bool {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}
	
	header = strings.ToLower(header)
	for _, h := range hopByHopHeaders {
		if strings.ToLower(h) == header {
			return true
		}
	}
	return false
}