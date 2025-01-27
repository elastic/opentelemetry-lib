package configelasticsearch

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
)

const defaultElasticsearchEnvName = "ELASTICSEARCH_URL"

var (
	errConfigEndpointRequired = errors.New("exactly one of [endpoint, endpoints, cloudid] must be specified")
	errConfigEmptyEndpoint    = errors.New("endpoint must not be empty")
)

type ClientConfig struct {
	// This setting is required if CloudID is not set and if the
	// ELASTICSEARCH_URL environment variable is not set.
	Endpoints []string `mapstructure:"endpoints"`

	// CloudID holds the cloud ID to identify the Elastic Cloud cluster to send events to.
	// https://www.elastic.co/guide/en/cloud/current/ec-cloud-id.html
	//
	// This setting is required if no URL is configured.
	CloudID string `mapstructure:"cloudid"`

	confighttp.ClientConfig `mapstructure:",squash"`

	// TelemetrySettings contains settings useful for testing/debugging purposes
	// This is experimental and may change at any time.
	TelemetrySettings `mapstructure:"telemetry"`

	Discovery DiscoverySettings `mapstructure:"discover"`
	Retry     RetrySettings     `mapstructure:"retry"`

	CacheDuration time.Duration `mapstructure:"cache_duration"`
}

type TelemetrySettings struct {
	LogRequestBody  bool `mapstructure:"log_request_body"`
	LogResponseBody bool `mapstructure:"log_response_body"`
}

// RetrySettings defines settings for the HTTP request retries in the Elasticsearch exporter.
// Failed sends are retried with exponential backoff.
type RetrySettings struct {
	// Enabled allows users to disable retry without having to comment out all settings.
	Enabled bool `mapstructure:"enabled"`

	// MaxRequests configures how often an HTTP request is attempted before it is assumed to be failed.
	// Deprecated: use MaxRetries instead.
	MaxRequests int `mapstructure:"max_requests"`

	// MaxRetries configures how many times an HTTP request is retried.
	MaxRetries int `mapstructure:"max_retries"`

	// InitialInterval configures the initial waiting time if a request failed.
	InitialInterval time.Duration `mapstructure:"initial_interval"`

	// MaxInterval configures the max waiting time if consecutive requests failed.
	MaxInterval time.Duration `mapstructure:"max_interval"`

	// RetryOnStatus configures the status codes that trigger request or document level retries.
	RetryOnStatus []int `mapstructure:"retry_on_status"`
}

// DiscoverySettings defines Elasticsearch node discovery related settings.
// The exporter will check Elasticsearch regularly for available nodes
// and updates the list of hosts if discovery is enabled. Newly discovered
// nodes will automatically be used for load balancing.
//
// DiscoverySettings should not be enabled when operating Elasticsearch behind a proxy
// or load balancer.
//
// https://www.elastic.co/blog/elasticsearch-sniffing-best-practices-what-when-why-how
type DiscoverySettings struct {
	// OnStart, if set, instructs the exporter to look for available Elasticsearch
	// nodes the first time the exporter connects to the cluster.
	OnStart bool `mapstructure:"on_start"`

	// Interval instructs the exporter to renew the list of Elasticsearch URLs
	// with the given interval. URLs will not be updated if Interval is <=0.
	Interval time.Duration `mapstructure:"interval"`
}

// Validate checks the receiver configuration is valid.
func (cfg *ClientConfig) Validate() error {
	endpoints, err := cfg.endpoints()
	if err != nil {
		return err
	}
	for _, endpoint := range endpoints {
		if err := validateEndpoint(endpoint); err != nil {
			return fmt.Errorf("invalid endpoint %q: %w", endpoint, err)
		}
	}

	if cfg.Retry.MaxRequests != 0 && cfg.Retry.MaxRetries != 0 {
		return errors.New("must not specify both retry::max_requests and retry::max_retries")
	}
	if cfg.Retry.MaxRequests < 0 {
		return errors.New("retry::max_requests should be non-negative")
	}
	if cfg.Retry.MaxRetries < 0 {
		return errors.New("retry::max_retries should be non-negative")
	}
	return nil
}

func validateEndpoint(endpoint string) error {
	if endpoint == "" {
		return errConfigEmptyEndpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	switch u.Scheme {
	case "http", "https":
	default:
		return fmt.Errorf(`invalid scheme %q, expected "http" or "https"`, u.Scheme)
	}
	return nil
}

func (cfg *ClientConfig) endpoints() ([]string, error) {
	// Exactly one of endpoint, endpoints, or cloudid must be configured.
	// If none are set, then $ELASTICSEARCH_URL may be specified instead.
	var endpoints []string
	var numEndpointConfigs int

	if cfg.Endpoint != "" {
		numEndpointConfigs++
		endpoints = []string{cfg.Endpoint}
	}
	if len(cfg.Endpoints) > 0 {
		numEndpointConfigs++
		endpoints = cfg.Endpoints
	}
	if cfg.CloudID != "" {
		numEndpointConfigs++
		u, err := parseCloudID(cfg.CloudID)
		if err != nil {
			return nil, err
		}
		endpoints = []string{u.String()}
	}
	if numEndpointConfigs == 0 {
		if v := os.Getenv(defaultElasticsearchEnvName); v != "" {
			numEndpointConfigs++
			endpoints = strings.Split(v, ",")
			for i, endpoint := range endpoints {
				endpoints[i] = strings.TrimSpace(endpoint)
			}
		}
	}
	if numEndpointConfigs != 1 {
		return nil, errConfigEndpointRequired
	}
	return endpoints, nil
}

// Based on "addrFromCloudID" in go-elasticsearch.
func parseCloudID(input string) (*url.URL, error) {
	_, after, ok := strings.Cut(input, ":")
	if !ok {
		return nil, fmt.Errorf("invalid CloudID %q", input)
	}

	decoded, err := base64.StdEncoding.DecodeString(after)
	if err != nil {
		return nil, err
	}

	before, after, ok := strings.Cut(string(decoded), "$")
	if !ok {
		return nil, fmt.Errorf("invalid decoded CloudID %q", string(decoded))
	}
	return url.Parse(fmt.Sprintf("https://%s.%s", after, before))
}
