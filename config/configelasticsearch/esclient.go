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
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/go-elasticsearch/v8"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

const defaultMaxRetries = 2

// clientLogger implements the estransport.Logger interface
// that is required by the Elasticsearch client for logging.
type clientLogger struct {
	*zap.Logger
	logRequestBody  bool
	logResponseBody bool
}

// LogRoundTrip should not modify the request or response, except for consuming and closing the body.
// Implementations have to check for nil values in request and response.
func (cl *clientLogger) LogRoundTrip(requ *http.Request, resp *http.Response, clientErr error, _ time.Time, dur time.Duration) error {
	zl := cl.Logger

	var fields []zap.Field
	if cl.logRequestBody && requ != nil && requ.Body != nil {
		body := requ.Body
		if requ.Header.Get("Content-Encoding") == "gzip" {
			if r, err := gzip.NewReader(body); err == nil {
				defer r.Close()
				body = r
			}
		}
		if b, err := io.ReadAll(body); err == nil {
			fields = append(fields, zap.ByteString("request_body", b))
		}
	}
	if cl.logResponseBody && resp != nil && resp.Body != nil {
		if b, err := io.ReadAll(resp.Body); err == nil {
			fields = append(fields, zap.ByteString("response_body", b))
		}
	}

	switch {
	case clientErr == nil && resp != nil:
		fields = append(
			fields,
			zap.String("path", requ.URL.Path),
			zap.String("method", requ.Method),
			zap.Duration("duration", dur),
			zap.String("status", resp.Status),
		)
		zl.Debug("Request roundtrip completed.", fields...)

	case clientErr != nil:
		fields = append(
			fields,
			zap.NamedError("reason", clientErr),
		)
		zl.Debug("Request failed.", fields...)
	}

	return nil
}

// RequestBodyEnabled makes the client pass a copy of request body to the logger.
func (cl *clientLogger) RequestBodyEnabled() bool {
	return cl.logRequestBody
}

// ResponseBodyEnabled makes the client pass a copy of response body to the logger.
func (cl *clientLogger) ResponseBodyEnabled() bool {
	return cl.logResponseBody
}

// user_agent should be added with the confighttp client
func (cfg *ClientConfig) ToClient(
	ctx context.Context,
	host component.Host,
	telemetry component.TelemetrySettings,
) (*elasticsearch.Client, error) {
	httpClient, err := cfg.ClientConfig.ToClient(ctx, host, telemetry)
	if err != nil {
		return nil, err
	}

	// endpoints converts Config.Endpoints, Config.CloudID,
	// and Config.ClientConfig.Endpoint to a list of addresses.
	endpoints, err := cfg.endpoints()
	if err != nil {
		return nil, err
	}

	esLogger := clientLogger{
		Logger:          telemetry.Logger,
		logRequestBody:  cfg.TelemetrySettings.LogRequestBody,
		logResponseBody: cfg.TelemetrySettings.LogResponseBody,
	}

	maxRetries := defaultMaxRetries
	if cfg.Retry.MaxRetries != 0 {
		maxRetries = cfg.Retry.MaxRetries
	}

	return elasticsearch.NewClient(elasticsearch.Config{
		Transport: httpClient.Transport,

		// configure connection setup
		Addresses: endpoints,

		// configure retry behavior
		RetryOnStatus: cfg.Retry.RetryOnStatus,
		DisableRetry:  !cfg.Retry.Enabled,
		// RetryOnError:  retryOnError, // should be used from esclient version 8 onwards
		MaxRetries:   maxRetries,
		RetryBackoff: createElasticsearchBackoffFunc(&cfg.Retry),

		// configure sniffing
		DiscoverNodesOnStart:  cfg.Discovery.OnStart,
		DiscoverNodesInterval: cfg.Discovery.Interval,

		// configure internal metrics reporting and logging
		EnableMetrics:     false, // TODO
		EnableDebugLogger: false, // TODO
		Logger:            &esLogger,

		Instrumentation: elasticsearch.NewOpenTelemetryInstrumentation(telemetry.TracerProvider, false),
	})
}

func createElasticsearchBackoffFunc(config *RetrySettings) func(int) time.Duration {
	if !config.Enabled {
		return nil
	}

	expBackoff := backoff.NewExponentialBackOff()
	if config.InitialInterval > 0 {
		expBackoff.InitialInterval = config.InitialInterval
	}
	if config.MaxInterval > 0 {
		expBackoff.MaxInterval = config.MaxInterval
	}
	expBackoff.Reset()

	return func(attempts int) time.Duration {
		if attempts == 1 {
			expBackoff.Reset()
		}

		return expBackoff.NextBackOff()
	}
}
