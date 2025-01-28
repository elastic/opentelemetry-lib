// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package configelasticsearch

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		configFile string
		id         string
		expected   component.Config
	}{
		{
			id:         "multiple_endpoints",
			configFile: "config.yaml",
			expected: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.Endpoints = []string{
					"http://localhost:9200",
					"http://localhost:8080",
				}
			}),
		},
		{
			id:         "with_cloudid",
			configFile: "config.yaml",
			expected: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.CloudID = "foo:YmFyLmNsb3VkLmVzLmlvJGFiYzEyMyRkZWY0NTY="
			}),
		},
		{
			id:         "confighttp_endpoint",
			configFile: "config.yaml",
			expected: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.Endpoint = "https://elastic.example.com:9200"
			}),
		},
		{
			id:         "compression_none",
			configFile: "config.yaml",
			expected: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.Endpoint = "https://elastic.example.com:9200"

				cfg.Compression = "none"
			}),
		},
		{
			id:         "compression_gzip",
			configFile: "config.yaml",
			expected: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.Endpoint = "https://elastic.example.com:9200"

				cfg.Compression = "gzip"
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			cfg := NewDefaultClientConfig()

			cm, err := confmaptest.LoadConf(filepath.Join("testdata", tt.configFile))
			require.NoError(t, err)

			sub, err := cm.Sub(tt.id)
			require.NoError(t, err)
			require.NoError(t, sub.Unmarshal(&cfg))

			assert.NoError(t, component.ValidateConfig(cfg))
			assert.Equal(t, tt.expected, &cfg)

			_, err = cfg.ToClient(context.Background(), componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)
		})
	}
}

// TestConfig_Validate tests the error cases of Config.Validate.
//
// Successful validation should be covered by TestConfig above.
func TestConfig_Validate(t *testing.T) {
	tests := map[string]struct {
		config *ClientConfig
		err    string
	}{
		"no endpoints": {
			config: withDefaultConfig(),
			err:    "exactly one of [endpoint, endpoints, cloudid] must be specified",
		},
		"empty endpoint": {
			config: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.Endpoints = []string{""}
			}),
			err: `invalid endpoint "": endpoint must not be empty`,
		},
		"invalid endpoint": {
			config: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.Endpoints = []string{"*:!"}
			}),
			err: `invalid endpoint "*:!": parse "*:!": first path segment in URL cannot contain colon`,
		},
		"invalid cloudid": {
			config: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.CloudID = "invalid"
			}),
			err: `invalid CloudID "invalid"`,
		},
		"invalid base64 cloudid": {
			config: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.CloudID = "foo:_invalid_base64_characters"
			}),
			err: `illegal base64 data at input byte 0`,
		},
		"invalid decoded cloudid": {
			config: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.CloudID = "foo:YWJj"
			}),
			err: `invalid decoded CloudID "abc"`,
		},
		"endpoints and cloudid both set": {
			config: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.Endpoints = []string{"http://test:9200"}
				cfg.CloudID = "foo:YmFyLmNsb3VkLmVzLmlvJGFiYzEyMyRkZWY0NTY="
			}),
			err: "exactly one of [endpoint, endpoints, cloudid] must be specified",
		},
		"endpoint and endpoints both set": {
			config: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.Endpoint = "http://test:9200"
				cfg.Endpoints = []string{"http://test:9200"}
			}),
			err: "exactly one of [endpoint, endpoints, cloudid] must be specified",
		},
		"invalid scheme": {
			config: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.Endpoints = []string{"without_scheme"}
			}),
			err: `invalid endpoint "without_scheme": invalid scheme "", expected "http" or "https"`,
		},
		"compression unsupported": {
			config: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.Endpoints = []string{"http://test:9200"}
				cfg.Compression = configcompression.TypeSnappy
			}),
			err: `compression must be one of [none, gzip]`,
		},
		"invalid max_requests specified": {
			config: withDefaultConfig(func(cfg *ClientConfig) {
				cfg.Endpoints = []string{"http://test:9200"}
				cfg.Retry.MaxRetries = -1
			}),
			err: `retry::max_requests should be non-negative`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.EqualError(t, component.ValidateConfig(tt.config), tt.err)
		})
	}
}

func TestConfig_Validate_Environment(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		t.Setenv("ELASTICSEARCH_URL", "http://test:9200")
		config := withDefaultConfig()
		err := component.ValidateConfig(config)
		require.NoError(t, err)
	})
	t.Run("invalid", func(t *testing.T) {
		t.Setenv("ELASTICSEARCH_URL", "http://valid:9200, *:!")
		config := withDefaultConfig()
		err := component.ValidateConfig(config)
		assert.EqualError(t, err, `invalid endpoint "*:!": parse "*:!": first path segment in URL cannot contain colon`)
	})
}

func withDefaultConfig(fns ...func(*ClientConfig)) *ClientConfig {
	cfg := NewDefaultClientConfig()
	for _, fn := range fns {
		fn(&cfg)
	}
	return &cfg
}
