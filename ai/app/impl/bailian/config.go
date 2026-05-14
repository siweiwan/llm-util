package bailian

import (
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseUrl                 = "https://dashscope.aliyuncs.com/api/v1/apps"
	defaultEmptyMessagesLimit uint = 300
	defaultRetryTimes         int  = 3
	defaultTimeout                 = 10 * time.Minute
)

type ClientConfig struct {
	apiKey string

	AppID      string
	Workspace  string
	BaseURL    string
	HTTPClient *http.Client

	EmptyMessagesLimit uint
	RetryTimes         int
}

func NewClientConfig(appID, apiKey string, setters ...ConfigOption) ClientConfig {
	config := ClientConfig{
		apiKey:  apiKey,
		AppID:   appID,
		BaseURL: defaultBaseUrl,
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
		EmptyMessagesLimit: defaultEmptyMessagesLimit,
		RetryTimes:         defaultRetryTimes,
	}

	for _, setter := range setters {
		setter(&config)
	}
	return config
}

type ConfigOption func(*ClientConfig)

func WithAppID(appID string) ConfigOption {
	return func(config *ClientConfig) {
		config.AppID = appID
	}
}

func WithEmptyMessagesLimit(limit uint) ConfigOption {
	return func(config *ClientConfig) {
		config.EmptyMessagesLimit = limit
	}
}

func WithBaseUrl(url string) ConfigOption {
	return func(config *ClientConfig) {
		config.BaseURL = url
		if strings.HasSuffix(url, "/") {
			config.BaseURL = strings.TrimSuffix(url, "/")
		}
	}
}

func WithRetryTimes(retryTimes int) ConfigOption {
	return func(config *ClientConfig) {
		config.RetryTimes = retryTimes
	}
}

func WithTimeout(timeout time.Duration) ConfigOption {
	return func(config *ClientConfig) {
		config.HTTPClient.Timeout = timeout
	}
}

func WithHTTPClient(client *http.Client) ConfigOption {
	return func(config *ClientConfig) {
		config.HTTPClient = client
	}
}
