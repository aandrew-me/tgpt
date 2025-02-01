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
		tls_client.WithClientProfile(profiles.Firefox_110),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		// tls_client.WithInsecureSkipVerify(),
	}

	// proxy in environment variables
	proxyAddr := os.Getenv("HTTP_PROXY")
	if proxyAddr == "" {
		proxyAddr = os.Getenv("http_proxy")
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
		if strings.HasPrefix(proxyAddr, "http://") || strings.HasPrefix(proxyAddr, "socks5://") {
			options = append(options, tls_client.WithProxyUrl(proxyAddr))
		} else {
			if !strings.HasPrefix(proxyAddr, "#") {
				fmt.Fprintln(os.Stderr, "Warning: Invalid proxy format, must start with http:// or socks5://")
			}
		}
	}

	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
}
