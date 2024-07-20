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
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(120),
		tls_client.WithClientProfile(profiles.Firefox_110),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar),
		// tls_client.WithInsecureSkipVerify(),
	}

	proxyAddress := os.Getenv("HTTP_PROXY")
	if proxyAddress == "" {
		proxyAddress = os.Getenv("http_proxy")
	} else {
	}

	if proxyAddress != "" {
		if strings.HasPrefix(proxyAddress, "http://") || strings.HasPrefix(proxyAddress, "socks5://") {
			proxyOption := tls_client.WithProxyUrl(proxyAddress)
			options = append(options, proxyOption)
		}
	} else {
		homeDir, _ := os.UserHomeDir()

		proxyConfigLocations := []string{
			"proxy.txt",
			filepath.Join(homeDir, ".config", "tgpt", "proxy.txt"),
		}

		for _, proxyConfigLocation := range proxyConfigLocations {
			_, err := os.Stat(proxyConfigLocation)
			if err == nil {
				proxyConfig, readErr := os.ReadFile(proxyConfigLocation)
				if readErr != nil {
					fmt.Fprintln(os.Stderr, "Error reading file proxy.txt:", readErr)
					return nil, readErr
				}

				proxyAddress := strings.TrimSpace(string(proxyConfig))
				if proxyAddress != "" {
					if strings.HasPrefix(proxyAddress, "http://") || strings.HasPrefix(proxyAddress, "socks5://") {
						proxyOption := tls_client.WithProxyUrl(proxyAddress)
						options = append(options, proxyOption)
					}
				}

				break
			}
		}
	}

	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
}
