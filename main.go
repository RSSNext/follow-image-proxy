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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")
}

func setXForwardedForHeader(req *http.Request, realIP string) {
	req.Header.Set("X-Forwarded-For", realIP)
}
