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

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/collector/semconv/v1.25.0"
	"google.golang.org/grpc/codes"
)

// Tests the enrichment logic for elastic's transaction definition.
func TestElasticTransactionEnrich(t *testing.T) {
	for _, tc := range []struct {
		name          string
		input         ptrace.Span
		enrichedAttrs map[string]any
	}{
		{
			name:  "empty",
			input: ptrace.NewSpan(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "Success",
				AttributeTransactionType:   "unknown",
			},
		},
		{
			name: "http_status_ok",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.Attributes().PutInt(semconv.AttributeHTTPStatusCode, http.StatusOK)
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "HTTP 2xx",
				AttributeTransactionType:   "request",
			},
		},
		{
			name: "http_status_1xx",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				// attributes should be preferred over span status
				span.Status().SetCode(ptrace.StatusCodeOk)
				span.Attributes().PutInt(
					semconv.AttributeHTTPResponseStatusCode,
					http.StatusProcessing,
				)
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "HTTP 1xx",
				AttributeTransactionType:   "request",
			},
		},
		{
			name: "http_status_5xx",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.Attributes().PutInt(
					semconv.AttributeHTTPStatusCode, http.StatusInternalServerError)
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "failure",
				AttributeTransactionResult: "HTTP 5xx",
				AttributeTransactionType:   "request",
			},
		},
		{
			name: "grpc_status_ok",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				// attributes should be preferred over span status
				span.Status().SetCode(ptrace.StatusCodeOk)
				span.Attributes().PutInt(
					semconv.AttributeRPCGRPCStatusCode,
					int64(codes.OK),
				)
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "OK",
				AttributeTransactionType:   "request",
			},
		},
		{
			name: "grpc_status_internal_error",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				// attributes should be preferred over span status
				span.Status().SetCode(ptrace.StatusCodeOk)
				span.Attributes().PutInt(
					semconv.AttributeRPCGRPCStatusCode,
					int64(codes.Internal),
				)
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "Internal",
				AttributeTransactionType:   "request",
			},
		},
		{
			name: "span_status_ok",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.Status().SetCode(ptrace.StatusCodeOk)
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeTransactionResult: "Success",
				AttributeTransactionType:   "unknown",
			},
		},
		{
			name: "span_status_error",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.Status().SetCode(ptrace.StatusCodeError)
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "failure",
				AttributeTransactionResult: "Error",
				AttributeTransactionType:   "unknown",
			},
		},
		{
			name: "messaging_type_kafka",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.Attributes().PutStr(semconv.AttributeMessagingSystem, "kafka")
				return span
			}(),
			enrichedAttrs: map[string]any{
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

			var span Span
			span.Enrich(tc.input)

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
		enrichedAttrs map[string]any
	}{
		{
			name:  "empty",
			input: getElasticSpan(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome: "success",
			},
		},
		{
			name: "peer_service",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeServiceTargetName: "testsvc",
			},
		},
		{
			name: "http_span_basic",
			input: func() ptrace.Span {
				span := getElasticSpan()
				// peer-service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutInt(
					semconv.AttributeHTTPResponseStatusCode,
					http.StatusOK,
				)
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "http",
				AttributeServiceTargetName: "testsvc",
			},
		},
		{
			name: "http_span_full_url",
			input: func() ptrace.Span {
				span := getElasticSpan()
				// peer-service should be ignored if more specific deductions
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
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "http",
				AttributeServiceTargetName: "www.foo.bar:443",
			},
		},
		{
			name: "http_span_no_full_url",
			input: func() ptrace.Span {
				span := getElasticSpan()
				// peer-service should be ignored if more specific deductions
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
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "http",
				AttributeServiceTargetName: "www.foo.bar:443",
			},
		},
		{
			name: "rpc_span_grpc",
			input: func() ptrace.Span {
				span := getElasticSpan()
				// peer-service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutInt(
					semconv.AttributeRPCGRPCStatusCode,
					int64(codes.OK),
				)
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "grpc",
				AttributeServiceTargetName: "testsvc",
			},
		},
		{
			name: "rpc_span_system",
			input: func() ptrace.Span {
				span := getElasticSpan()
				// peer-service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv.AttributeRPCSystem, "xmlrpc")
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "xmlrpc",
				AttributeServiceTargetName: "testsvc",
			},
		},
		{
			name: "rpc_span_service",
			input: func() ptrace.Span {
				span := getElasticSpan()
				// peer-service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv.AttributeRPCService, "service.Test")
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "external",
				AttributeServiceTargetName: "service.Test",
			},
		},
		{
			name: "messaging_basic",
			input: func() ptrace.Span {
				span := getElasticSpan()
				// peer-service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv.AttributeMessagingSystem, "kafka")
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "kafka",
				AttributeServiceTargetName: "testsvc",
			},
		},
		{
			name: "messaging_destination",
			input: func() ptrace.Span {
				span := getElasticSpan()
				// peer-service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv.AttributeMessagingDestinationName, "t1")
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "messaging",
				AttributeServiceTargetName: "t1",
			},
		},
		{
			name: "messaging_temp_destination",
			input: func() ptrace.Span {
				span := getElasticSpan()
				// peer-service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv.AttributePeerService, "testsvc")
				span.Attributes().PutBool(semconv.AttributeMessagingDestinationTemporary, true)
				span.Attributes().PutStr(semconv.AttributeMessagingDestinationName, "t1")
				return span
			}(),
			enrichedAttrs: map[string]any{
				AttributeEventOutcome:      "success",
				AttributeServiceTargetType: "messaging",
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

			var span Span
			span.Enrich(tc.input)

			assert.Empty(t, cmp.Diff(expectedAttrs, tc.input.Attributes().AsRaw()))
		})
	}
}
