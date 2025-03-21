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
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"github.com/elastic/opentelemetry-lib/elasticattr"
	"github.com/elastic/opentelemetry-lib/enrichments/trace/config"
	"github.com/google/go-cmp/cmp"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/ptracetest"
	"github.com/stretchr/testify/assert"
	"github.com/ua-parser/uap-go/uaparser"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv25 "go.opentelemetry.io/collector/semconv/v1.25.0"
	semconv27 "go.opentelemetry.io/collector/semconv/v1.27.0"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc/codes"
)

// Tests the enrichment logic for elastic's transaction definition.
func TestElasticTransactionEnrich(t *testing.T) {
	now := time.Unix(3600, 0)
	expectedDuration := time.Minute
	endTs := pcommon.NewTimestampFromTime(now)
	startTs := pcommon.NewTimestampFromTime(now.Add(-1 * expectedDuration))
	getElasticTxn := func() ptrace.Span {
		span := ptrace.NewSpan()
		span.SetSpanID([8]byte{1})
		span.SetStartTimestamp(startTs)
		span.SetEndTimestamp(endTs)
		return span
	}
	for _, tc := range []struct {
		name              string
		input             ptrace.Span
		config            config.ElasticTransactionConfig
		enrichedAttrs     map[string]any
		expectedSpanLinks *ptrace.SpanLinkSlice
	}{
		{
			// test case gives a summary of what is emitted by default
			name:   "empty",
			input:  ptrace.NewSpan(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    int64(0),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "",
				elasticattr.TransactionName:                "",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          int64(0),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.TransactionResult:              "Success",
				elasticattr.TransactionType:                "unknown",
			},
		},
		{
			name:          "all_disabled",
			input:         getElasticTxn(),
			enrichedAttrs: map[string]any{},
		},
		{
			name: "with_pvalue",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.TraceState().FromRaw("ot=p:8;")
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    int64(0),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "",
				elasticattr.TransactionName:                "",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(256),
				elasticattr.TransactionDurationUs:          int64(0),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(256),
				elasticattr.TransactionResult:              "Success",
				elasticattr.TransactionType:                "unknown",
			},
		},
		{
			name: "http_status_ok",
			input: func() ptrace.Span {
				span := getElasticTxn()
				span.SetName("testtxn")
				span.Attributes().PutInt(semconv25.AttributeHTTPStatusCode, http.StatusOK)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionName:                "testtxn",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.TransactionResult:              "HTTP 2xx",
				elasticattr.TransactionType:                "request",
			},
		},
		{
			name: "http_status_1xx",
			input: func() ptrace.Span {
				span := getElasticTxn()
				span.SetName("testtxn")
				span.SetSpanID([8]byte{1})
				// attributes should be preferred over span status for txn result
				span.Status().SetCode(ptrace.StatusCodeOk)
				span.Attributes().PutInt(
					semconv25.AttributeHTTPResponseStatusCode,
					http.StatusProcessing,
				)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionName:                "testtxn",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.TransactionResult:              "HTTP 1xx",
				elasticattr.TransactionType:                "request",
			},
		},
		{
			name: "http_status_5xx",
			input: func() ptrace.Span {
				span := getElasticTxn()
				span.SetName("testtxn")
				span.SetSpanID([8]byte{1})
				// span status code should take precedence over http status attributes
				// for setting event.outcome
				span.Status().SetCode(ptrace.StatusCodeOk)
				span.Attributes().PutInt(
					semconv25.AttributeHTTPStatusCode, http.StatusInternalServerError)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionName:                "testtxn",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.TransactionResult:              "HTTP 5xx",
				elasticattr.TransactionType:                "request",
			},
		},
		{
			name: "grpc_status_ok",
			input: func() ptrace.Span {
				span := getElasticTxn()
				span.SetName("testtxn")
				span.SetSpanID([8]byte{1})
				// attributes should be preferred over span status for txn result
				span.Status().SetCode(ptrace.StatusCodeOk)
				span.Attributes().PutInt(
					semconv25.AttributeRPCGRPCStatusCode,
					int64(codes.OK),
				)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionName:                "testtxn",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.TransactionResult:              "OK",
				elasticattr.TransactionType:                "request",
			},
		},
		{
			name: "grpc_status_internal_error",
			input: func() ptrace.Span {
				span := getElasticTxn()
				span.SetName("testtxn")
				span.SetSpanID([8]byte{1})
				// attributes should be preferred over span status for txn result
				span.Status().SetCode(ptrace.StatusCodeOk)
				span.Attributes().PutInt(
					semconv25.AttributeRPCGRPCStatusCode,
					int64(codes.Internal),
				)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionName:                "testtxn",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.TransactionResult:              "Internal",
				elasticattr.TransactionType:                "request",
			},
		},
		{
			name: "span_status_ok",
			input: func() ptrace.Span {
				span := getElasticTxn()
				span.SetName("testtxn")
				span.SetSpanID([8]byte{1})
				span.Status().SetCode(ptrace.StatusCodeOk)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionName:                "testtxn",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.TransactionResult:              "Success",
				elasticattr.TransactionType:                "unknown",
			},
		},
		{
			name: "span_status_error",
			input: func() ptrace.Span {
				span := getElasticTxn()
				span.SetName("testtxn")
				span.SetSpanID([8]byte{1})
				span.Status().SetCode(ptrace.StatusCodeError)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionName:                "testtxn",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "failure",
				elasticattr.SuccessCount:                   int64(0),
				elasticattr.TransactionResult:              "Error",
				elasticattr.TransactionType:                "unknown",
			},
		},
		{
			name: "messaging_type_kafka",
			input: func() ptrace.Span {
				span := getElasticTxn()
				span.SetName("testtxn")
				span.SetSpanID([8]byte{1})
				span.Attributes().PutStr(semconv25.AttributeMessagingSystem, "kafka")
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionName:                "testtxn",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.TransactionResult:              "Success",
				elasticattr.TransactionType:                "messaging",
			},
		},
		{
			name: "inferred_spans",
			input: func() ptrace.Span {
				span := getElasticTxn()
				span.SetName("testtxn")
				span.SetSpanID([8]byte{1})
				normalLink := span.Links().AppendEmpty()
				normalLink.SetSpanID([8]byte{2})

				childLink := span.Links().AppendEmpty()
				childLink.SetSpanID([8]byte{3})
				childLink.Attributes().PutBool("is_child", true)

				childLink2 := span.Links().AppendEmpty()
				childLink2.SetSpanID([8]byte{4})
				childLink2.Attributes().PutBool("elastic.is_child", true)
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionName:                "testtxn",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.TransactionResult:              "Success",
				elasticattr.TransactionType:                "unknown",
				elasticattr.ChildIDs:                       []any{"0300000000000000", "0400000000000000"},
			},
			expectedSpanLinks: func() *ptrace.SpanLinkSlice {
				spanLinks := ptrace.NewSpanLinkSlice()
				// Only the span link without `is_child` or `elastic.is_child` is expected
				spanLinks.AppendEmpty().SetSpanID([8]byte{2})
				return &spanLinks
			}(),
		},
		{
			name: "user_agent_parse_name_version",
			input: func() ptrace.Span {
				span := getElasticTxn()
				span.SetName("testtxn")
				span.Attributes().PutStr(semconv27.AttributeUserAgentOriginal, "Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionName:                "testtxn",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.TransactionResult:              "Success",
				elasticattr.TransactionType:                "unknown",
				semconv27.AttributeUserAgentName:           "Firefox",
				semconv27.AttributeUserAgentVersion:        "126.0",
			},
		},
		{
			name: "user_agent_no_override",
			input: func() ptrace.Span {
				span := getElasticTxn()
				span.SetName("testtxn")
				span.Attributes().PutStr(semconv27.AttributeUserAgentOriginal, "Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")
				// In practical situations the user_agent.{name, version} should be derived from the
				// original user agent, however, for testing we are setting different values.
				span.Attributes().PutStr(semconv27.AttributeUserAgentName, "Chrome")
				span.Attributes().PutStr(semconv27.AttributeUserAgentVersion, "51.0.2704")
				return span
			}(),
			config: config.Enabled().Transaction,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionName:                "testtxn",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.TransactionResult:              "Success",
				elasticattr.TransactionType:                "unknown",
				// If user_agent.{name, version} are already set then don't override them.
				semconv27.AttributeUserAgentName:    "Chrome",
				semconv27.AttributeUserAgentVersion: "51.0.2704",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expectedSpan := ptrace.NewSpan()
			tc.input.CopyTo(expectedSpan)

			// Merge with the expected attributes and override the span links.
			for k, v := range tc.enrichedAttrs {
				expectedSpan.Attributes().PutEmpty(k).FromRaw(v)
			}
			// Override span links
			if tc.expectedSpanLinks != nil {
				tc.expectedSpanLinks.CopyTo(expectedSpan.Links())
			} else {
				expectedSpan.Links().RemoveIf(func(_ ptrace.SpanLink) bool { return true })
			}

			EnrichSpan(tc.input, config.Config{
				Transaction: tc.config,
			}, uaparser.NewFromSaved())
			assert.NoError(t, ptracetest.CompareSpan(expectedSpan, tc.input))
		})
	}
}

// Tests root spans that represent a dependency and are mapped to a transaction.
func TestRootSpanAsDependencyEnrich(t *testing.T) {
	for _, tc := range []struct {
		name              string
		input             ptrace.Span
		config            config.Config
		enrichedAttrs     map[string]any
		expectedSpanLinks *ptrace.SpanLinkSlice
	}{
		{
			name: "outgoing_http_root_span",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetName("rootClientSpan")
				span.SetSpanID([8]byte{1})
				span.SetKind(ptrace.SpanKindClient)
				span.Attributes().PutStr(semconv27.AttributeHTTPRequestMethod, "GET")
				span.Attributes().PutStr(semconv27.AttributeURLFull, "http://localhost:8080")
				span.Attributes().PutInt(semconv27.AttributeHTTPResponseStatusCode, 200)
				span.Attributes().PutStr(semconv27.AttributeNetworkProtocolVersion, "1.1")
				return span
			}(),
			config: config.Enabled(),
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    int64(0),
				elasticattr.TransactionName:                "rootClientSpan",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.SpanType:                       "external",
				elasticattr.SpanSubtype:                    "http",
				elasticattr.SpanDestinationServiceResource: "localhost:8080",
				elasticattr.SpanName:                       "rootClientSpan",
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetName:              "localhost:8080",
				elasticattr.ServiceTargetType:              "http",
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionDurationUs:          int64(0),
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionResult:              "HTTP 2xx",
				elasticattr.TransactionType:                "external.http",
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.SpanDurationUs:                 int64(0),
				elasticattr.SpanRepresentativeCount:        float64(1),
			},
		},
		{
			name: "db_root_span",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetName("rootClientSpan")
				span.SetSpanID([8]byte{1})
				span.SetKind(ptrace.SpanKindClient)
				span.Attributes().PutStr(semconv25.AttributeDBSystem, "mssql")

				span.Attributes().PutStr(semconv25.AttributeDBName, "myDb")
				span.Attributes().PutStr(semconv25.AttributeDBOperation, "SELECT")
				span.Attributes().PutStr(semconv25.AttributeDBStatement, "SELECT * FROM wuser_table")
				return span
			}(),
			config: config.Enabled(),
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    int64(0),
				elasticattr.TransactionName:                "rootClientSpan",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.SpanType:                       "db",
				elasticattr.SpanSubtype:                    "mssql",
				elasticattr.SpanDestinationServiceResource: "mssql",
				elasticattr.SpanName:                       "rootClientSpan",
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetName:              "myDb",
				elasticattr.ServiceTargetType:              "mssql",
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionDurationUs:          int64(0),
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionResult:              "Success",
				elasticattr.TransactionType:                "db.mssql",
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.SpanDurationUs:                 int64(0),
				elasticattr.SpanRepresentativeCount:        float64(1),
			},
		},
		{
			name: "producer_messaging_span",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetName("rootClientSpan")
				span.SetSpanID([8]byte{1})
				span.SetKind(ptrace.SpanKindProducer)

				span.Attributes().PutStr(semconv25.AttributeServerAddress, "myServer")
				span.Attributes().PutStr(semconv25.AttributeServerPort, "1234")
				span.Attributes().PutStr(semconv25.AttributeMessagingSystem, "rabbitmq")
				span.Attributes().PutStr(semconv25.AttributeMessagingDestinationName, "T")
				span.Attributes().PutStr(semconv25.AttributeMessagingOperation, "publish")
				span.Attributes().PutStr(semconv25.AttributeMessagingClientID, "a")
				return span
			}(),
			config: config.Enabled(),
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    int64(0),
				elasticattr.TransactionName:                "rootClientSpan",
				elasticattr.ProcessorEvent:                 "transaction",
				elasticattr.SpanType:                       "messaging",
				elasticattr.SpanSubtype:                    "rabbitmq",
				elasticattr.SpanDestinationServiceResource: "rabbitmq/T",
				elasticattr.SpanName:                       "rootClientSpan",
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetName:              "T",
				elasticattr.ServiceTargetType:              "rabbitmq",
				elasticattr.TransactionID:                  "0100000000000000",
				elasticattr.TransactionDurationUs:          int64(0),
				elasticattr.TransactionRepresentativeCount: float64(1),
				elasticattr.TransactionResult:              "Success",
				elasticattr.TransactionType:                "messaging.rabbitmq",
				elasticattr.TransactionSampled:             true,
				elasticattr.TransactionRoot:                true,
				elasticattr.SpanDurationUs:                 int64(0),
				elasticattr.SpanRepresentativeCount:        float64(1),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expectedSpan := ptrace.NewSpan()
			tc.input.CopyTo(expectedSpan)

			// Merge with the expected attributes and override the span links.
			for k, v := range tc.enrichedAttrs {
				expectedSpan.Attributes().PutEmpty(k).FromRaw(v)
			}
			// Override span links
			if tc.expectedSpanLinks != nil {
				tc.expectedSpanLinks.CopyTo(expectedSpan.Links())
			} else {
				expectedSpan.Links().RemoveIf(func(_ ptrace.SpanLink) bool { return true })
			}

			EnrichSpan(tc.input, tc.config, uaparser.NewFromSaved())
			assert.NoError(t, ptracetest.CompareSpan(expectedSpan, tc.input))
		})
	}
}

// Tests the enrichment logic for elastic's span definition.
func TestElasticSpanEnrich(t *testing.T) {
	now := time.Unix(3600, 0)
	expectedDuration := time.Minute
	endTs := pcommon.NewTimestampFromTime(now)
	startTs := pcommon.NewTimestampFromTime(now.Add(-1 * expectedDuration))
	getElasticSpan := func() ptrace.Span {
		span := ptrace.NewSpan()
		span.SetParentSpanID([8]byte{8, 9, 10, 11, 12, 13, 14})
		span.SetStartTimestamp(startTs)
		span.SetEndTimestamp(endTs)
		return span
	}
	for _, tc := range []struct {
		name              string
		input             ptrace.Span
		config            config.ElasticSpanConfig
		enrichedAttrs     map[string]any
		expectedSpanLinks *ptrace.SpanLinkSlice
	}{
		{
			// test case gives a summary of what is emitted by default
			name: "empty",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetParentSpanID([8]byte{1})
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:             int64(0),
				elasticattr.SpanName:                "",
				elasticattr.ProcessorEvent:          "span",
				elasticattr.SpanRepresentativeCount: float64(1),
				elasticattr.SpanType:                "unknown",
				elasticattr.SpanDurationUs:          int64(0),
				elasticattr.EventOutcome:            "success",
				elasticattr.SuccessCount:            int64(1),
			},
		},
		{
			name:          "all_disabled",
			input:         getElasticSpan(),
			enrichedAttrs: map[string]any{},
		},
		{
			name: "internal_span",
			input: func() ptrace.Span {
				span := ptrace.NewSpan()
				span.SetParentSpanID([8]byte{1})
				span.SetKind(ptrace.SpanKindInternal)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:             int64(0),
				elasticattr.SpanName:                "",
				elasticattr.ProcessorEvent:          "span",
				elasticattr.SpanRepresentativeCount: float64(1),
				elasticattr.SpanType:                "app",
				elasticattr.SpanSubtype:             "internal",
				elasticattr.SpanDurationUs:          int64(0),
				elasticattr.EventOutcome:            "success",
				elasticattr.SuccessCount:            int64(1),
			},
		},
		{
			name: "peer_service",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "unknown",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetName:              "testsvc",
				elasticattr.ServiceTargetType:              "",
				elasticattr.SpanDestinationServiceResource: "testsvc",
			},
		},
		{
			name: "http_span_basic",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutInt(
					semconv25.AttributeHTTPResponseStatusCode,
					http.StatusOK,
				)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "external",
				elasticattr.SpanSubtype:                    "http",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "http",
				elasticattr.ServiceTargetName:              "testsvc",
				elasticattr.SpanDestinationServiceResource: "testsvc",
			},
		},
		{
			name: "http_span_full_url",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				// peer.service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutInt(
					semconv25.AttributeHTTPResponseStatusCode,
					http.StatusOK,
				)
				span.Attributes().PutStr(
					semconv25.AttributeURLFull,
					"https://www.foo.bar:443/search?q=OpenTelemetry#SemConv",
				)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "external",
				elasticattr.SpanSubtype:                    "http",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "http",
				elasticattr.ServiceTargetName:              "www.foo.bar:443",
				elasticattr.SpanDestinationServiceResource: "testsvc",
			},
		},
		{
			name: "http_span_deprecated_http_url",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				// peer.service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutInt(
					semconv25.AttributeHTTPResponseStatusCode,
					http.StatusOK,
				)
				span.Attributes().PutStr(
					semconv25.AttributeHTTPURL,
					"https://www.foo.bar:443/search?q=OpenTelemetry#SemConv",
				)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "external",
				elasticattr.SpanSubtype:                    "http",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "http",
				elasticattr.ServiceTargetName:              "www.foo.bar:443",
				elasticattr.SpanDestinationServiceResource: "testsvc",
			},
		},
		{
			name: "http_span_no_full_url",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				// peer.service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutInt(
					semconv25.AttributeHTTPResponseStatusCode,
					http.StatusOK,
				)
				span.Attributes().PutStr(semconv25.AttributeURLDomain, "www.foo.bar")
				span.Attributes().PutInt(semconv25.AttributeURLPort, 443)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "external",
				elasticattr.SpanSubtype:                    "http",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "http",
				elasticattr.ServiceTargetName:              "www.foo.bar:443",
				elasticattr.SpanDestinationServiceResource: "testsvc",
			},
		},
		{
			name: "rpc_span_grpc",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutInt(
					semconv25.AttributeRPCGRPCStatusCode,
					int64(codes.OK),
				)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "external",
				elasticattr.SpanSubtype:                    "grpc",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "grpc",
				elasticattr.ServiceTargetName:              "testsvc",
				elasticattr.SpanDestinationServiceResource: "testsvc",
			},
		},
		{
			name: "rpc_span_system",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv25.AttributeRPCSystem, "xmlrpc")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "external",
				elasticattr.SpanSubtype:                    "xmlrpc",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "xmlrpc",
				elasticattr.ServiceTargetName:              "testsvc",
				elasticattr.SpanDestinationServiceResource: "testsvc",
			},
		},
		{
			name: "rpc_span_service",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				// peer.service should be ignored if more specific deductions
				// can be made about the service target.
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv25.AttributeRPCService, "service.Test")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "external",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "external",
				elasticattr.ServiceTargetName:              "service.Test",
				elasticattr.SpanDestinationServiceResource: "testsvc",
			},
		},
		{
			name: "rpc_span_service.{address, port}",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				// No peer.service is set
				span.Attributes().PutStr(semconv25.AttributeRPCService, "service.Test")
				span.Attributes().PutStr(semconv25.AttributeServerAddress, "10.2.20.18")
				span.Attributes().PutInt(semconv25.AttributeServerPort, 8081)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "external",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "external",
				elasticattr.ServiceTargetName:              "service.Test",
				elasticattr.SpanDestinationServiceResource: "10.2.20.18:8081",
			},
		},
		{
			name: "rpc_span_net.peer.{address, port}_fallback",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				// No peer.service is set
				span.Attributes().PutStr(semconv25.AttributeRPCService, "service.Test")
				span.Attributes().PutStr(semconv25.AttributeNetPeerName, "10.2.20.18")
				span.Attributes().PutInt(semconv25.AttributeNetPeerPort, 8081)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "external",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "external",
				elasticattr.ServiceTargetName:              "service.Test",
				elasticattr.SpanDestinationServiceResource: "10.2.20.18:8081",
			},
		},
		{
			name: "messaging_basic",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv25.AttributeMessagingSystem, "kafka")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "messaging",
				elasticattr.SpanSubtype:                    "kafka",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "kafka",
				elasticattr.ServiceTargetName:              "testsvc",
				elasticattr.SpanDestinationServiceResource: "testsvc",
			},
		},
		{
			name: "messaging_destination",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutStr(semconv25.AttributeMessagingDestinationName, "t1")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "messaging",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "messaging",
				elasticattr.ServiceTargetName:              "t1",
				elasticattr.SpanDestinationServiceResource: "testsvc/t1",
			},
		},
		{
			name: "messaging_temp_destination",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutBool(semconv25.AttributeMessagingDestinationTemporary, true)
				span.Attributes().PutStr(semconv25.AttributeMessagingDestinationName, "t1")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "messaging",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "messaging",
				elasticattr.ServiceTargetName:              "testsvc",
				elasticattr.SpanDestinationServiceResource: "testsvc/t1",
			},
		},
		{
			name: "db_over_http",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutStr(
					semconv25.AttributeURLFull,
					"https://localhost:9200/index/_search?q=user.id:kimchy",
				)
				span.Attributes().PutStr(
					semconv25.AttributeDBSystem,
					semconv25.AttributeDBSystemElasticsearch,
				)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "db",
				elasticattr.SpanSubtype:                    "elasticsearch",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "elasticsearch",
				elasticattr.ServiceTargetName:              "testsvc",
				elasticattr.SpanDestinationServiceResource: "testsvc",
			},
		},
		{
			name: "db_over_rpc",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.Attributes().PutStr(semconv25.AttributePeerService, "testsvc")
				span.Attributes().PutStr(
					semconv25.AttributeRPCSystem,
					semconv25.AttributeRPCSystemGRPC,
				)
				span.Attributes().PutStr(semconv25.AttributeRPCService, "cassandra.API")
				span.Attributes().PutStr(
					semconv25.AttributeRPCGRPCStatusCode,
					semconv25.AttributeRPCGRPCStatusCodeOk,
				)
				span.Attributes().PutStr(
					semconv25.AttributeDBSystem,
					semconv25.AttributeDBSystemCassandra,
				)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "db",
				elasticattr.SpanSubtype:                    "cassandra",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "cassandra",
				elasticattr.ServiceTargetName:              "testsvc",
				elasticattr.SpanDestinationServiceResource: "testsvc",
			},
		},
		{
			name: "inferred_spans",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.SetSpanID([8]byte{1})
				normalLink := span.Links().AppendEmpty()
				normalLink.SetSpanID([8]byte{2})

				childLink := span.Links().AppendEmpty()
				childLink.SetSpanID([8]byte{3})
				childLink.Attributes().PutBool("is_child", true)

				childLink2 := span.Links().AppendEmpty()
				childLink2.SetSpanID([8]byte{4})
				childLink2.Attributes().PutBool("elastic.is_child", true)
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:             startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                "testspan",
				elasticattr.ProcessorEvent:          "span",
				elasticattr.SpanRepresentativeCount: float64(1),
				elasticattr.SpanType:                "unknown",
				elasticattr.SpanDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:            "success",
				elasticattr.SuccessCount:            int64(1),
				elasticattr.ChildIDs:                []any{"0300000000000000", "0400000000000000"},
			},
			expectedSpanLinks: func() *ptrace.SpanLinkSlice {
				spanLinks := ptrace.NewSpanLinkSlice()
				// Only the span link without `is_child` or `elastic.is_child` is expected
				spanLinks.AppendEmpty().SetSpanID([8]byte{2})
				return &spanLinks
			}(),
		},
		{
			name: "genai_with_system",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.SetSpanID([8]byte{1})
				span.Attributes().PutStr(semconv27.AttributeGenAiSystem, "openai")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:             startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                "testspan",
				elasticattr.ProcessorEvent:          "span",
				elasticattr.SpanRepresentativeCount: float64(1),
				elasticattr.SpanType:                "genai",
				elasticattr.SpanSubtype:             "openai",
				elasticattr.SpanDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:            "success",
				elasticattr.SuccessCount:            int64(1),
			},
		},
		{
			name: "rpc_span_with_only_rpc_sevice_attr",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				// rpc.service should be used as destination.service.resource
				// if no other attributes are present.
				span.Attributes().PutStr(semconv25.AttributeRPCService, "myService")
				span.Attributes().PutStr(semconv25.AttributeRPCSystem, "grpc")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:                    startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                       "testspan",
				elasticattr.ProcessorEvent:                 "span",
				elasticattr.SpanRepresentativeCount:        float64(1),
				elasticattr.SpanType:                       "external",
				elasticattr.SpanSubtype:                    "grpc",
				elasticattr.SpanDurationUs:                 expectedDuration.Microseconds(),
				elasticattr.EventOutcome:                   "success",
				elasticattr.SuccessCount:                   int64(1),
				elasticattr.ServiceTargetType:              "grpc",
				elasticattr.ServiceTargetName:              "myService",
				elasticattr.SpanDestinationServiceResource: "myService",
			},
		},
		{
			name: "user_agent_parse_name_version",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.SetSpanID([8]byte{1})
				span.Attributes().PutStr(semconv27.AttributeUserAgentOriginal, "Mozilla/5.0 (iPhone; CPU iPhone OS 13_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1.1 Mobile/15E148 Safari/604.1")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:             startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                "testspan",
				elasticattr.ProcessorEvent:          "span",
				elasticattr.SpanRepresentativeCount: float64(1),
				elasticattr.SpanType:                "unknown",
				elasticattr.SpanDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:            "success",
				elasticattr.SuccessCount:            int64(1),
				semconv27.AttributeUserAgentName:    "Mobile Safari",
				semconv27.AttributeUserAgentVersion: "13.1.1",
			},
		},
		{
			name: "user_agent_no_override",
			input: func() ptrace.Span {
				span := getElasticSpan()
				span.SetName("testspan")
				span.SetSpanID([8]byte{1})
				span.Attributes().PutStr(semconv27.AttributeUserAgentOriginal, "Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")
				// In practical situations the user_agent.{name, version} should be derived from the
				// original user agent, however, for testing we are setting different values.
				span.Attributes().PutStr(semconv27.AttributeUserAgentName, "Chrome")
				span.Attributes().PutStr(semconv27.AttributeUserAgentVersion, "51.0.2704")
				return span
			}(),
			config: config.Enabled().Span,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:             startTs.AsTime().UnixMicro(),
				elasticattr.SpanName:                "testspan",
				elasticattr.ProcessorEvent:          "span",
				elasticattr.SpanRepresentativeCount: float64(1),
				elasticattr.SpanType:                "unknown",
				elasticattr.SpanDurationUs:          expectedDuration.Microseconds(),
				elasticattr.EventOutcome:            "success",
				elasticattr.SuccessCount:            int64(1),
				// If user_agent.{name, version} are already set then don't override them.
				semconv27.AttributeUserAgentName:    "Chrome",
				semconv27.AttributeUserAgentVersion: "51.0.2704",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expectedSpan := ptrace.NewSpan()
			tc.input.CopyTo(expectedSpan)

			// Merge with the expected attributes and override the span links.
			for k, v := range tc.enrichedAttrs {
				expectedSpan.Attributes().PutEmpty(k).FromRaw(v)
			}
			// Override span links
			if tc.expectedSpanLinks != nil {
				tc.expectedSpanLinks.CopyTo(expectedSpan.Links())
			} else {
				expectedSpan.Links().RemoveIf(func(_ ptrace.SpanLink) bool { return true })
			}

			EnrichSpan(tc.input, config.Config{
				Span: tc.config,
			}, uaparser.NewFromSaved())
			assert.NoError(t, ptracetest.CompareSpan(expectedSpan, tc.input))
		})
	}
}

func TestSpanEventEnrich(t *testing.T) {
	now := time.Unix(3600, 0)
	ts := pcommon.NewTimestampFromTime(now)
	for _, tc := range []struct {
		name          string
		parent        ptrace.Span
		input         ptrace.SpanEvent
		config        config.SpanEventConfig
		errorID       bool // indicates if the error ID should be present in the result
		enrichedAttrs map[string]any
	}{
		{
			name:   "not_exception",
			parent: ptrace.NewSpan(),
			input: func() ptrace.SpanEvent {
				event := ptrace.NewSpanEvent()
				event.SetTimestamp(ts)
				return event
			}(),
			config:  config.Enabled().SpanEvent,
			errorID: false, // error ID is only present for exceptions
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs: ts.AsTime().UnixMicro(),
			},
		},
		{
			name: "exception_with_elastic_txn",
			parent: func() ptrace.Span {
				// No parent, elastic txn
				span := ptrace.NewSpan()
				return span
			}(),
			input: func() ptrace.SpanEvent {
				event := ptrace.NewSpanEvent()
				event.SetName("exception")
				event.SetTimestamp(ts)
				event.Attributes().PutStr(semconv25.AttributeExceptionType, "java.net.ConnectionError")
				event.Attributes().PutStr(semconv25.AttributeExceptionMessage, "something is wrong")
				event.Attributes().PutStr(semconv25.AttributeExceptionStacktrace, `Exception in thread "main" java.lang.RuntimeException: Test exception\\n at com.example.GenerateTrace.methodB(GenerateTrace.java:13)\\n at com.example.GenerateTrace.methodA(GenerateTrace.java:9)\\n at com.example.GenerateTrace.main(GenerateTrace.java:5)`)
				return event
			}(),
			config:  config.Enabled().SpanEvent,
			errorID: true,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:           ts.AsTime().UnixMicro(),
				elasticattr.ProcessorEvent:        "error",
				elasticattr.ErrorExceptionHandled: true,
				elasticattr.ErrorGroupingKey: func() string {
					hash := md5.New()
					hash.Write([]byte("java.net.ConnectionError"))
					return hex.EncodeToString(hash.Sum(nil))
				}(),
				elasticattr.ErrorGroupingName:  "something is wrong",
				elasticattr.TransactionSampled: true,
				elasticattr.TransactionType:    "unknown",
			},
		},
		{
			name: "exception_with_elastic_span",
			parent: func() ptrace.Span {
				// Parent, elastic span
				span := ptrace.NewSpan()
				span.SetParentSpanID([8]byte{8, 9, 10, 11, 12, 13, 14})
				return span
			}(),
			input: func() ptrace.SpanEvent {
				event := ptrace.NewSpanEvent()
				event.SetName("exception")
				event.SetTimestamp(ts)
				event.Attributes().PutStr(semconv25.AttributeExceptionType, "java.net.ConnectionError")
				event.Attributes().PutStr(semconv25.AttributeExceptionMessage, "something is wrong")
				event.Attributes().PutStr(semconv25.AttributeExceptionStacktrace, `Exception in thread "main" java.lang.RuntimeException: Test exception\\n at com.example.GenerateTrace.methodB(GenerateTrace.java:13)\\n at com.example.GenerateTrace.methodA(GenerateTrace.java:9)\\n at com.example.GenerateTrace.main(GenerateTrace.java:5)`)
				return event
			}(),
			config:  config.Enabled().SpanEvent,
			errorID: true,
			enrichedAttrs: map[string]any{
				elasticattr.TimestampUs:           ts.AsTime().UnixMicro(),
				elasticattr.ProcessorEvent:        "error",
				elasticattr.ErrorExceptionHandled: true,
				elasticattr.ErrorGroupingKey: func() string {
					hash := md5.New()
					hash.Write([]byte("java.net.ConnectionError"))
					return hex.EncodeToString(hash.Sum(nil))
				}(),
				elasticattr.ErrorGroupingName: "something is wrong",
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

			tc.input.MoveTo(tc.parent.Events().AppendEmpty())
			EnrichSpan(tc.parent, config.Config{
				SpanEvent: tc.config,
			}, uaparser.NewFromSaved())

			actual := tc.parent.Events().At(0).Attributes()
			errorID, ok := actual.Get(elasticattr.ErrorID)
			assert.Equal(t, tc.errorID, ok, "error_id must be present for exception and must not be present for non-exception")
			if tc.errorID {
				assert.NotEmpty(t, errorID, "error_id must not be empty")
			}
			// Ignore error in actual diff since it is randomly generated
			actual.Remove(elasticattr.ErrorID)
			assert.Empty(t, cmp.Diff(expectedAttrs, actual.AsRaw()))
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
