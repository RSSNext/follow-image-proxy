package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

var (
	client *http.Client
	once   sync.Once
)

func main() {
	e := echo.New()
	e.GET("/", handleProxyRequest)
	e.Start(":8080")
}

func getHTTPClient() *http.Client {
	once.Do(func() {
		client = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	})
	return client
}

func handleProxyRequest(c echo.Context) error {
	targetURL := c.QueryParam("url")
	if targetURL == "" {
		return c.String(http.StatusBadRequest, "Missing 'url' query parameter")
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Invalid URL: %v", err))
	}

	req, err := http.NewRequestWithContext(c.Request().Context(), "GET", targetURL, nil)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error creating request: %v", err))
	}

	setRefererHeader(req, parsedURL)
	setUserAgentHeader(req)
	setXForwardedForHeader(req, c.RealIP())
	setAdditionalHeaders(req, parsedURL)

	resp, err := getHTTPClient().Do(req)
	if err != nil {
		return c.String(http.StatusBadGateway, fmt.Sprintf("Error fetching the URL: %v", err))
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "image") {
		return c.String(http.StatusForbidden, fmt.Sprintf("Forbidden: Content-Type is not an image. Received Content-Type: %s", contentType))
	}

	c.Response().Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Response().WriteHeader(resp.StatusCode)

	_, err = io.Copy(c.Response().Writer, resp.Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Error copying response body: %v", err))
	}
	return nil
}

func setRefererHeader(req *http.Request, parsedURL *url.URL) {
	hostnameParts := strings.Split(parsedURL.Hostname(), ".")
	if len(hostnameParts) > 2 {
		mainDomain := strings.Join(hostnameParts[len(hostnameParts)-2:], ".")
		req.Header.Set("Referer", fmt.Sprintf("https://%s", mainDomain))
	}
}

func setUserAgentHeader(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
}

func setXForwardedForHeader(req *http.Request, realIP string) {
	req.Header.Set("X-Forwarded-For", realIP)
}

func setAdditionalHeaders(req *http.Request, parsedURL *url.URL) {
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Fetch-Dest", "image")
	req.Header.Set("Sec-Fetch-Mode", "no-cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Origin", fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host))
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
}
