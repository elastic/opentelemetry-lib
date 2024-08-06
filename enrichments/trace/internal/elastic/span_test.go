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

package elastic

import (
	"net/http"
	"testing"

	"github.com/elastic/opentelemetry-lib/enrichments/trace/config"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/collector/semconv/v1.25.0"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc/codes"
)

// Tests the enrichment logic for elastic's transaction definition.
func TestElasticTransactionEnrich(t *testing.T) {
	for _, tc := range []struct {
		name          string
		input         ptrace.Span
		config        config.ElasticTransactionConfig
		enrichedAttrs map[string]any
	}{
		{
			name:   "empty",
			input:  ptrace.NewSpan(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				AttributeTraceRoot:         true,
				AttributeTransactionName:   "",
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "Success",
				AttributeTransactionType:   "unknown",
			},
		},
		{
			name:          "all_disabled",
			input:         ptrace.NewSpan(),
			enrichedAttrs: map[string]any{},
		},
		{
			name: "http_status_ok",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetName("testtxn")
				span.Attributes().PutInt(semconv.AttributeHTTPStatusCode, http.StatusOK)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				AttributeTraceRoot:         true,
				AttributeTransactionName:   "testtxn",
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "HTTP 2xx",
				AttributeTransactionType:   "request",
			},
		},
		{
			name: "http_status_1xx",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetName("testtxn")
				// attributes should be preferred over span status for txn result
				span.Status().SetCode(ptrace.StatusCodeOk)
				span.Attributes().PutInt(
					semconv.AttributeHTTPResponseStatusCode,
					http.StatusProcessing,
				)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				AttributeTraceRoot:         true,
				AttributeTransactionName:   "testtxn",
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "HTTP 1xx",
				AttributeTransactionType:   "request",
			},
		},
		{
			name: "http_status_5xx",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetName("testtxn")
				// span status code should take precedence over http status attributes
				// for setting event.outcome
				span.Status().SetCode(ptrace.StatusCodeOk)
				span.Attributes().PutInt(
					semconv.AttributeHTTPStatusCode, http.StatusInternalServerError)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				AttributeTraceRoot:         true,
				AttributeTransactionName:   "testtxn",
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "HTTP 5xx",
				AttributeTransactionType:   "request",
			},
		},
		{
			name: "grpc_status_ok",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetName("testtxn")
				// attributes should be preferred over span status for txn result
				span.Status().SetCode(ptrace.StatusCodeOk)
				span.Attributes().PutInt(
					semconv.AttributeRPCGRPCStatusCode,
					int64(codes.OK),
				)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				AttributeTraceRoot:         true,
				AttributeTransactionName:   "testtxn",
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "OK",
				AttributeTransactionType:   "request",
			},
		},
		{
			name: "grpc_status_internal_error",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetName("testtxn")
				// attributes should be preferred over span status for txn result
				span.Status().SetCode(ptrace.StatusCodeOk)
				span.Attributes().PutInt(
					semconv.AttributeRPCGRPCStatusCode,
					int64(codes.Internal),
				)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				AttributeTraceRoot:         true,
				AttributeTransactionName:   "testtxn",
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "Internal",
				AttributeTransactionType:   "request",
			},
		},
		{
			name: "span_status_ok",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetName("testtxn")
				span.Status().SetCode(ptrace.StatusCodeOk)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				AttributeTraceRoot:         true,
				AttributeTransactionName:   "testtxn",
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "Success",
				AttributeTransactionType:   "unknown",
			},
		},
		{
			name: "span_status_error",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetName("testtxn")
				span.Status().SetCode(ptrace.StatusCodeError)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				AttributeTraceRoot:         true,
				AttributeTransactionName:   "testtxn",
				AttributeEventOutcome:      "failure",
				AttributeTransactionResult: "Error",
				AttributeTransactionType:   "unknown",
			},
		},
		{
			name: "messaging_type_kafka",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetName("testtxn")
				span.Attributes().PutStr(semconv.AttributeMessagingSystem, "kafka")
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				AttributeTraceRoot:         true,
				AttributeTransactionName:   "testtxn",
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "Success",
				AttributeTransactionType:   "messaging",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Merge existing input attrs with the attrs added
			// by enrichment to get the expected attributes.
			expectedAttrs := tc.input.Attributes().AsRaw()
			for k, v := range tc.enrichedAttrs {
				expectedAttrs[k] = v
			}

			EnrichSpan(tc.input, config.Config{
				Transaction: tc.config,
			})

			assert.Empty(t, cmp.Diff(expectedAttrs, tc.input.Attributes().AsRaw()))
		})
	}
}

// Tests the enrichment logic for elastic's span definition.
func TestElasticSpanEnrich(t *testing.T) {
	getElasticSpan := func() ptrace.Span {
		span := ptrace.NewSpan()
		span.SetParentSpanID([8]byte{8, 9, 10, 11, 12, 13, 14})
		return span
	}
	for _, tc := range []struct {
		name          string
		input         ptrace.Span
		config        config.ElasticSpanConfig
		enrichedAttrs map[string]any
	}{
		{
			name:   "empty",
			input:  getElasticSpan(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:     "",
				AttributeEventOutcome: "success",
			},
		},
		{
			name:          "all_disabled",
			input:         getElasticSpan(),
			enrichedAttrs: map[string]any{},
		},
		{
			name: "peer_service",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetName: "testsvc",
				AttributeServiceTargetType: "",
			},
		},
		{
			name: "http_span_basic",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutInt(
					semconv.AttributeHTTPResponseStatusCode,
					http.StatusOK,
				)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "http",
				AttributeServiceTargetName: "testsvc",
			},
		},
		{
			name: "http_span_full_url",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				// peer.service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutInt(
					semconv.AttributeHTTPResponseStatusCode,
					http.StatusOK,
				)
				span.Attributes().PutStr(
					semconv.AttributeURLFull,
					"https://www.foo.bar:443/search?q=OpenTelemetry#SemConv",
				)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "http",
				AttributeServiceTargetName: "www.foo.bar:443",
			},
		},
		{
			name: "http_span_no_full_url",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				// peer.service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutInt(
					semconv.AttributeHTTPResponseStatusCode,
					http.StatusOK,
				)
				span.Attributes().PutStr(semconv.AttributeURLDomain, "www.foo.bar")
				span.Attributes().PutInt(semconv.AttributeURLPort, 443)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "http",
				AttributeServiceTargetName: "www.foo.bar:443",
			},
		},
		{
			name: "rpc_span_grpc",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutInt(
					semconv.AttributeRPCGRPCStatusCode,
					int64(codes.OK),
				)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "grpc",
				AttributeServiceTargetName: "testsvc",
			},
		},
		{
			name: "rpc_span_system",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv.AttributeRPCSystem, "xmlrpc")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "xmlrpc",
				AttributeServiceTargetName: "testsvc",
			},
		},
		{
			name: "rpc_span_service",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				// peer.service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv.AttributeRPCService, "service.Test")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "external",
				AttributeServiceTargetName: "service.Test",
			},
		},
		{
			name: "messaging_basic",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv.AttributeMessagingSystem, "kafka")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "kafka",
				AttributeServiceTargetName: "testsvc",
			},
		},
		{
			name: "messaging_destination",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv.AttributeMessagingDestinationName, "t1")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "messaging",
				AttributeServiceTargetName: "t1",
			},
		},
		{
			name: "messaging_temp_destination",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutBool(semconv.AttributeMessagingDestinationTemporary, true)
				span.Attributes().PutStr(semconv.AttributeMessagingDestinationName, "t1")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "messaging",
				AttributeServiceTargetName: "testsvc",
			},
		},
		{
			name: "db_over_http",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutStr(
					semconv.AttributeURLFull,
					"https://localhost:9200/index/_search?q=user.id:kimchy",
				)
				span.Attributes().PutStr(
					semconv.AttributeDBSystem,
					semconv.AttributeDBSystemElasticsearch,
				)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "elasticsearch",
				AttributeServiceTargetName: "testsvc",
			},
		},
		{
			name: "db_over_rpc",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutStr(
					semconv.AttributeRPCSystem,
					semconv.AttributeRPCSystemGRPC,
				)
				span.Attributes().PutStr(semconv.AttributeRPCService, "cassandra.API")
				span.Attributes().PutStr(
					semconv.AttributeRPCGRPCStatusCode,
					semconv.AttributeRPCGRPCStatusCodeOk,
				)
				span.Attributes().PutStr(
					semconv.AttributeDBSystem,
					semconv.AttributeDBSystemCassandra,
				)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				AttributeSpanName:          "testspan",
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "cassandra",
				AttributeServiceTargetName: "testsvc",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Merge existing input attrs with the attrs added
			// by enrichment to get the expected attributes.
			expectedAttrs := tc.input.Attributes().AsRaw()
			for k, v := range tc.enrichedAttrs {
				expectedAttrs[k] = v
			}

			EnrichSpan(tc.input, config.Config{
				Span: tc.config,
			})

			assert.Empty(t, cmp.Diff(expectedAttrs, tc.input.Attributes().AsRaw()))
		})
	}
}

func TestIsElasticTransaction(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input ptrace.Span
		isTxn bool
	}{
		{
			name:  "no_parent_span",
			input: ptrace.NewSpan(),
			isTxn: true,
		},
		{
			name: "parent_span",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetParentSpanID([8]byte{8, 9, 10, 11, 12, 13, 14})
				return span
			}(),
			isTxn: false,
		},
		{
			name: "remote_parent_span",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetParentSpanID([8]byte{8, 9, 10, 11, 12, 13, 14})
				flags := tracepb.SpanFlags_SPAN_FLAGS_CONTEXT_HAS_IS_REMOTE_MASK
				flags = flags | tracepb.SpanFlags_SPAN_FLAGS_CONTEXT_IS_REMOTE_MASK
				span.SetFlags(uint32(flags))
				return span
			}(),
			isTxn: true,
		},
		{
			name: "local_parent_span",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetParentSpanID([8]byte{8, 9, 10, 11, 12, 13, 14})
				flags := tracepb.SpanFlags_SPAN_FLAGS_CONTEXT_HAS_IS_REMOTE_MASK
				span.SetFlags(uint32(flags))
				return span
			}(),
			isTxn: false,
		},
		{
			name: "unknown_parent_span_kind_server",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetParentSpanID([8]byte{8, 9, 10, 11, 12, 13, 14})
				span.SetKind(ptrace.SpanKindServer)
				return span
			}(),
			isTxn: true,
		},
		{
			name: "unknown_parent_span_kind_consumer",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetParentSpanID([8]byte{8, 9, 10, 11, 12, 13, 14})
				span.SetKind(ptrace.SpanKindConsumer)
				return span
			}(),
			isTxn: true,
		},
		{
			name: "unknown_parent_span_kind_producer",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetParentSpanID([8]byte{8, 9, 10, 11, 12, 13, 14})
				span.SetKind(ptrace.SpanKindProducer)
				return span
			}(),
			isTxn: false,
		},
		{
			name: "unknown_parent_span_kind_unspecified",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetParentSpanID([8]byte{8, 9, 10, 11, 12, 13, 14})
				return span
			}(),
			isTxn: false,
		},
		{
			name: "unknown_parent_span_kind_internal",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetParentSpanID([8]byte{8, 9, 10, 11, 12, 13, 14})
				span.SetKind(ptrace.SpanKindInternal)
				return span
			}(),
			isTxn: false,
		},
	} {
		assert.Equal(t, tc.isTxn, isElasticTransaction(tc.input))
	}
}
