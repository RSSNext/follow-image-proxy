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
		return c.String(http.StatusBadRequest, "Invalid URL")
	}

	req, err := http.NewRequestWithContext(c.Request().Context(), "GET", targetURL, nil)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error creating request")
	}

	setRefererHeader(req, parsedURL)

	resp, err := getHTTPClient().Do(req)
	if err != nil {
		return c.String(http.StatusBadGateway, "Error fetching the URL")
	}
	defer resp.Body.Close()

	if !strings.Contains(resp.Header.Get("Content-Type"), "image") {
		return c.String(http.StatusForbidden, "Forbidden")
	}

	c.Response().Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Response().WriteHeader(resp.StatusCode)

	_, err = io.Copy(c.Response().Writer, resp.Body)
	return err
}

func setRefererHeader(req *http.Request, parsedURL *url.URL) {
	hostnameParts := strings.Split(parsedURL.Hostname(), ".")
	if len(hostnameParts) > 2 {
		mainDomain := strings.Join(hostnameParts[len(hostnameParts)-2:], ".")
		req.Header.Set("Referer", fmt.Sprintf("https://%s", mainDomain))
	}
}
