package client

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

func GetProxyURL() string {
	proxyAddr := os.Getenv("HTTP_PROXY")
	if proxyAddr == "" {
		proxyAddr = os.Getenv("http_proxy")
	}
	if proxyAddr == "" {
		proxyAddr = os.Getenv("HTTPS_PROXY")
		if proxyAddr == "" {
			proxyAddr = os.Getenv("https_proxy")
		}
	}
	if proxyAddr == "" {
		proxyAddr = os.Getenv("ALL_PROXY")
		if proxyAddr == "" {
			proxyAddr = os.Getenv("all_proxy")
		}
	}

	if proxyAddr == "" {
		homeDir, _ := os.UserHomeDir()
		proxyFiles := []string{
			"proxy.txt",
			filepath.Join(homeDir, ".config", "tgpt", "proxy.txt"),
		}

		for _, file := range proxyFiles {
			if content, err := os.ReadFile(file); err == nil {
				candidate := strings.TrimSpace(string(content))
				if candidate != "" && !strings.HasPrefix(candidate, "#") {
					proxyAddr = candidate
					break
				}
			}
		}
	}

	return proxyAddr
}

func NewClient(timeoutSeconds ...int) (tls_client.HttpClient, error) {
	timeout := 600
	if len(timeoutSeconds) > 0 && timeoutSeconds[0] > 0 {
		timeout = timeoutSeconds[0]
	}

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(timeout),
		tls_client.WithDialer(net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}),
		// Allow overriding TLS fingerprint via env; default stays Firefox_117.
		tls_client.WithClientProfile(func() profiles.ClientProfile {
			p := profiles.Firefox_148
			switch strings.ToLower(os.Getenv("TLS_CLIENT_PROFILE")) {
			case "firefox_133", "ff133":
				p = profiles.Firefox_133
			case "firefox_117", "ff117", "":
				// keep default
			}
			return p
		}()),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		// tls_client.WithInsecureSkipVerify(),
	}

	proxyAddr := GetProxyURL()
	if proxyAddr != "" {
		if strings.HasPrefix(proxyAddr, "http://") || strings.HasPrefix(proxyAddr, "socks5://") || strings.HasPrefix(proxyAddr, "socks5h://") {
			options = append(options, tls_client.WithProxyUrl(proxyAddr))
		} else {
			if !strings.HasPrefix(proxyAddr, "#") {
				fmt.Fprintf(os.Stderr, "Warning: Invalid proxy format %q, must start with http://, socks5://, or socks5h://\n", proxyAddr)
			}
		}
	}

	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
}

func NewStandardHTTPClient(timeoutSeconds ...int) *http.Client {
	timeout := 600 * time.Second
	if len(timeoutSeconds) > 0 && timeoutSeconds[0] > 0 {
		timeout = time.Duration(timeoutSeconds[0]) * time.Second
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	proxyAddr := GetProxyURL()
	if proxyAddr != "" {
		if strings.HasPrefix(proxyAddr, "http://") || strings.HasPrefix(proxyAddr, "socks5://") || strings.HasPrefix(proxyAddr, "socks5h://") {
			if parsedURL, err := url.Parse(proxyAddr); err == nil {
				transport.Proxy = http.ProxyURL(parsedURL)
			}
		}
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
