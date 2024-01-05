package client

import (
	"fmt"
	"os"
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

	_, err := os.Stat("proxy.txt")
	if err == nil {
		proxyConfig, readErr := os.ReadFile("proxy.txt")
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
	}

	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
}
