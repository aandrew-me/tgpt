package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

func NewClient() (tls_client.HttpClient, error) {
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(600),
		// Allow overriding TLS fingerprint via env; default stays Firefox_117.
		tls_client.WithClientProfile(func() profiles.ClientProfile {
			p := profiles.Firefox_117
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

	// proxy in environment variables
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

	// No Proxy in env, try to load from configuration
	if proxyAddr == "" {
		homeDir, _ := os.UserHomeDir()
		proxyFiles := []string{
			"proxy.txt",
			filepath.Join(homeDir, ".config", "tgpt", "proxy.txt"),
		}

		for _, file := range proxyFiles {
			if content, err := os.ReadFile(file); err == nil {
				proxyAddr = strings.TrimSpace(string(content))
				break
			}
		}
	}

	// Set proxy options if valid proxy detected.
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
