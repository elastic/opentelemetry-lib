// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package configelasticsearch

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/confighttp"
)

const defaultElasticsearchEnvName = "ELASTICSEARCH_URL"

var (
	errConfigEndpointRequired = errors.New("exactly one of [endpoint, endpoints, cloudid] must be specified")
	errConfigEmptyEndpoint    = errors.New("endpoint must not be empty")
)

// NewDefaultClientConfig returns ClientConfig type object with
// the default values of 'MaxIdleConns' and 'IdleConnTimeout', as well as [http.DefaultTransport] values.
// Other config options are not added as they are initialized with 'zero value' by GoLang as default.
// We encourage to use this function to create an object of ClientConfig.
func NewDefaultClientConfig() ClientConfig {
	// The default values are taken from the values of 'DefaultTransport' of 'http' package.
	defaultHTTPClientConfig := confighttp.NewDefaultClientConfig()
	defaultHTTPClientConfig.Timeout = 90 * time.Second
	defaultHTTPClientConfig.Compression = configcompression.TypeGzip

	return ClientConfig{
		ClientConfig: defaultHTTPClientConfig,
		TelemetrySettings: TelemetrySettings{
			LogRequestBody:  false,
			LogResponseBody: false,
		},
		Retry: RetrySettings{
			Enabled:         true,
			MaxRetries:      0, // default is set in exporter code
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     1 * time.Minute,
			RetryOnStatus: []int{
				http.StatusTooManyRequests,
			},
		},
	}
}

type ClientConfig struct {
	confighttp.ClientConfig `mapstructure:",squash"`

	// CloudID holds the cloud ID to identify the Elastic Cloud cluster to send events to.
	// https://www.elastic.co/guide/en/cloud/current/ec-cloud-id.html
	//
	// This setting is required if no URL is configured.
	CloudID string `mapstructure:"cloudid"`
	// ELASTICSEARCH_URL environment variable is not set.
	Endpoints []string `mapstructure:"endpoints"`

	Retry RetrySettings `mapstructure:"retry"`

	Discovery DiscoverySettings `mapstructure:"discover"`

	// TelemetrySettings contains settings useful for testing/debugging purposes
	// This is experimental and may change at any time.
	TelemetrySettings `mapstructure:"telemetry"`
}

type TelemetrySettings struct {
	LogRequestBody  bool `mapstructure:"log_request_body"`
	LogResponseBody bool `mapstructure:"log_response_body"`
}

// RetrySettings defines settings for the HTTP request retries in the Elasticsearch exporter.
// Failed sends are retried with exponential backoff.
type RetrySettings struct {
	// RetryOnStatus configures the status codes that trigger request or document level retries.
	RetryOnStatus []int `mapstructure:"retry_on_status"`
	// MaxRetries configures how many times an HTTP request is retried.
	MaxRetries int `mapstructure:"max_retries"`
	// InitialInterval configures the initial waiting time if a request failed.
	InitialInterval time.Duration `mapstructure:"initial_interval"`
	// MaxInterval configures the max waiting time if consecutive requests failed.
	MaxInterval time.Duration `mapstructure:"max_interval"`
	// Enabled allows users to disable retry without having to comment out all settings.
	Enabled bool `mapstructure:"enabled"`
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

	if cfg.Compression != "none" && cfg.Compression != configcompression.TypeGzip {
		return errors.New("compression must be one of [none, gzip]")
	}

	if cfg.Retry.MaxRetries < 0 {
		return errors.New("retry::max_requests should be non-negative")
	}
	return cfg.ClientConfig.Validate()
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
