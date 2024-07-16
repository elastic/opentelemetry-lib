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
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/collector/semconv/v1.25.0"
	"google.golang.org/grpc/codes"
)

// Span is a representation of OTel `ptrace.Span`. The struct is used
// to enrich a span with Elastic specific attributes. These attributes
// are derived from the base attributes and appended to the span
// attributes. The enrichment logic is performed by categorizing the
// OTel spans into 2 different categories:
//   - Elastic transactions, defined as spans which measure the highest
//     level of work being performed with a service.
//   - Elastic spans, defined as all spans (including transactions).
//     However, for the enrichment logic spans are treated as a separate
//     entity i.e. all transactions are not enriched as spans and vice versa.
type Span struct {
	urlFull *url.URL

	peerService     string
	urlScheme       string
	urlDomain       string
	urlPath         string
	urlQuery        string
	rpcSystem       string
	rpcService      string
	grpcStatus      string
	dbName          string
	dbType          string
	messagingSystem string
	messagingDestn  string

	urlPort        int64
	httpStatusCode int64

	spanStatusCode ptrace.StatusCode

	isMessaging        bool
	isRPC              bool
	isHTTP             bool
	isDB               bool
	messagingDestnTemp bool
}

func (s *Span) Enrich(span ptrace.Span) {
	// Extract top level span information.
	s.spanStatusCode = span.Status().Code()

	// Extract information from span attributes.
	span.Attributes().Range(func(k string, v pcommon.Value) bool {
		switch k {
		case semconv.AttributePeerService:
			s.peerService = v.Str()
		case semconv.AttributeMessagingDestinationName:
			s.isMessaging = true
			s.messagingDestn = v.Str()
		case semconv.AttributeMessagingOperation:
			s.isMessaging = true
		case semconv.AttributeMessagingSystem:
			s.isMessaging = true
			s.messagingSystem = v.Str()
		case semconv.AttributeMessagingDestinationTemporary:
			s.isMessaging = true
			s.messagingDestnTemp = true
		case semconv.AttributeHTTPStatusCode,
			semconv.AttributeHTTPResponseStatusCode:
			s.isHTTP = true
			s.httpStatusCode = v.Int()
		case semconv.AttributeHTTPMethod,
			semconv.AttributeHTTPRequestMethod,
			semconv.AttributeHTTPURL,
			semconv.AttributeHTTPTarget,
			semconv.AttributeHTTPScheme,
			semconv.AttributeHTTPFlavor,
			semconv.AttributeNetHostName:
			s.isHTTP = true
		case semconv.AttributeURLFull:
			s.isHTTP = true
			// ignoring error as if parse fails then we don't want the url anyway
			s.urlFull, _ = url.Parse(v.Str())
		case semconv.AttributeURLScheme:
			s.isHTTP = true
			s.urlScheme = v.Str()
		case semconv.AttributeURLDomain:
			s.isHTTP = true
			s.urlDomain = v.Str()
		case semconv.AttributeURLPort:
			s.isHTTP = true
			s.urlPort = v.Int()
		case semconv.AttributeURLPath:
			s.isHTTP = true
			s.urlPath = v.Str()
		case semconv.AttributeURLQuery:
			s.isHTTP = true
			s.urlQuery = v.Str()
		case semconv.AttributeRPCGRPCStatusCode:
			s.isRPC = true
			s.grpcStatus = codes.Code(v.Int()).String()
		case semconv.AttributeRPCSystem:
			s.isRPC = true
			s.rpcSystem = v.Str()
		case semconv.AttributeRPCService:
			s.isRPC = true
			s.rpcService = v.Str()
		case semconv.AttributeDBStatement,
			semconv.AttributeDBUser:
			s.isDB = true
		case semconv.AttributeDBName:
			s.isDB = true
			s.dbName = v.Str()
		case semconv.AttributeDBSystem:
			s.isDB = true
			s.dbType = v.Str()
		}
		return true
	})

	// Ensure all dependent attributes are handled.
	s.normalizeAttributes()

	// Enrich the span depending on the nature of the span.
	if isElasticTransaction(span) {
		s.setTxnType(span)
		s.setTxnResult(span)
		s.setEventOutcome(span)
	} else {
		s.setEventOutcome(span)
		s.setServiceTarget(span)
	}
}

// normalizeAttributes sets any dependent attributes that
// might not have been explicitly set as an attribute.
func (s *Span) normalizeAttributes() {
	if s.rpcSystem == "" && s.grpcStatus != "" {
		s.rpcSystem = "grpc"
	}
}

func (s *Span) setTxnType(span ptrace.Span) {
	txnType := "unknown"
	switch {
	case s.isMessaging:
		txnType = "messaging"
		span.Attributes().PutStr("transaction.type", "messaging")
	case s.isRPC, s.isHTTP:
		txnType = "request"
	}
	span.Attributes().PutStr(AttributeTransactionType, txnType)
}

func (s *Span) setTxnResult(span ptrace.Span) {
	var result string

	if s.isHTTP && s.httpStatusCode > 0 {
		switch i := s.httpStatusCode / 100; i {
		case 1, 2, 3, 4, 5:
			result = standardStatusCodeResults[i-1]
		default:
			result = fmt.Sprintf("HTTP %d", s.httpStatusCode)
		}
	}
	if s.isRPC {
		result = s.grpcStatus
	}
	if result == "" {
		switch s.spanStatusCode {
		case ptrace.StatusCodeError:
			result = "Error"
		default:
			// default to success if all else fails
			result = "Success"
		}
	}

	span.Attributes().PutStr(AttributeTransactionResult, result)
}

func (s *Span) setEventOutcome(span ptrace.Span) {
	// default to success outcome
	outcome := "success"
	switch {
	case s.spanStatusCode == ptrace.StatusCodeError:
		outcome = "failure"
	case s.httpStatusCode >= http.StatusInternalServerError:
		// TODO (lahsivjar): Handle GRPC status code? - not handled in apm-data
		// TODO (lahsivjar): Move to HTTPResponseStatusCode? Backward compatibility?
		outcome = "failure"
	}
	span.Attributes().PutStr(AttributeEventOutcome, outcome)
}

func (s *Span) setServiceTarget(span ptrace.Span) {
	var targetType, targetName string

	if s.peerService != "" {
		targetName = s.peerService
	}

	switch {
	case s.isDB:
		targetType = "db"
		if s.dbType != "" {
			targetType = s.dbType
		}
		if s.dbName != "" {
			targetName = s.dbName
		}
	case s.isMessaging:
		targetType = "messaging"
		if s.messagingSystem != "" {
			targetType = s.messagingSystem
		}
		if !s.messagingDestnTemp && s.messagingDestn != "" {
			targetName = s.messagingDestn
		}
	case s.isRPC:
		targetType = "external"
		if s.rpcSystem != "" {
			targetType = s.rpcSystem
		}
		if s.rpcService != "" {
			targetName = s.rpcService
		}
	case s.isHTTP:
		targetType = "http"
		if resource := getHostPort(s.urlFull, s.urlDomain, s.urlPort); resource != "" {
			targetName = resource
		}
	}

	if targetType != "" {
		span.Attributes().PutStr(AttributeServiceTargetType, targetType)
	}
	if targetName != "" {
		span.Attributes().PutStr(AttributeServiceTargetName, targetName)
	}
}

func isElasticTransaction(span ptrace.Span) bool {
	if span.ParentSpanID().IsEmpty() ||
		span.Kind() == ptrace.SpanKindServer ||
		span.Kind() == ptrace.SpanKindConsumer {
		return true
	}
	return false
}

func getHostPort(urlFull *url.URL, urlDomain string, urlPort int64) string {
	if urlFull != nil {
		return urlFull.Host
	}
	if urlDomain != "" {
		if urlPort == 0 {
			return urlDomain
		}
		return net.JoinHostPort(urlDomain, strconv.FormatInt(urlPort, 10))
	}
	return ""
}

var standardStatusCodeResults = [...]string{
	"HTTP 1xx",
	"HTTP 2xx",
	"HTTP 3xx",
	"HTTP 4xx",
	"HTTP 5xx",
}
