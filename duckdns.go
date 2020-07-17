package main

import (
	"fmt"
	"net/http"
	"time"
)

const urlTemplate = "https://www.duckdns.org/update?domains=%s&token=%s"

var c = &http.Client{
	Timeout: 10 * time.Second,
}

// PingDuckDNS tells DuckDNS our IP address
// This is documented on their site: https://www.duckdns.org/install.jsp
func PingDuckDNS(site, token string) (err error) {
	resp, err := c.Get(fmt.Sprintf(urlTemplate, site, token))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("unexpected error status code %d", resp.StatusCode)
	}

	return
}
