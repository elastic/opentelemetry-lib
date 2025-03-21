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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/elastic/opentelemetry-lib/elasticattr"
	"github.com/elastic/opentelemetry-lib/enrichments/trace/config"
	"github.com/ua-parser/uap-go/uaparser"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv25 "go.opentelemetry.io/collector/semconv/v1.25.0"
	semconv27 "go.opentelemetry.io/collector/semconv/v1.27.0"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc/codes"
)

// EnrichSpan adds Elastic specific attributes to the OTel span.
// These attributes are derived from the base attributes and appended to
// the span attributes. The enrichment logic is performed by categorizing
// the OTel spans into 2 different categories:
//   - Elastic transactions, defined as spans which measure the highest
//     level of work being performed with a service.
//   - Elastic spans, defined as all spans (including transactions).
//     However, for the enrichment logic spans are treated as a separate
//     entity i.e. all transactions are not enriched as spans and vice versa.
func EnrichSpan(
	span ptrace.Span,
	cfg config.Config,
	userAgentParser *uaparser.Parser,
) {
	var c spanEnrichmentContext
	c.Enrich(span, cfg, userAgentParser)
}

type spanEnrichmentContext struct {
	urlFull *url.URL

	peerService              string
	serverAddress            string
	urlScheme                string
	urlDomain                string
	urlPath                  string
	urlQuery                 string
	rpcSystem                string
	rpcService               string
	grpcStatus               string
	dbName                   string
	dbSystem                 string
	messagingSystem          string
	messagingDestinationName string
	genAiSystem              string

	// The inferred* attributes are derived from a base attribute
	userAgentOriginal        string
	userAgentName            string
	userAgentVersion         string
	inferredUserAgentName    string
	inferredUserAgentVersion string

	serverPort     int64
	urlPort        int64
	httpStatusCode int64

	spanStatusCode ptrace.StatusCode

	// TODO (lahsivjar): Refactor span enrichment to better utilize isTransaction
	isTransaction            bool
	isMessaging              bool
	isRPC                    bool
	isHTTP                   bool
	isDB                     bool
	messagingDestinationTemp bool
	isGenAi                  bool
}

func (s *spanEnrichmentContext) Enrich(
	span ptrace.Span,
	cfg config.Config,
	userAgentParser *uaparser.Parser,
) {
	// Extract top level span information.
	s.spanStatusCode = span.Status().Code()

	// Extract information from span attributes.
	span.Attributes().Range(func(k string, v pcommon.Value) bool {
		switch k {
		case semconv25.AttributePeerService:
			s.peerService = v.Str()
		case semconv25.AttributeServerAddress:
			s.serverAddress = v.Str()
		case semconv25.AttributeServerPort:
			s.serverPort = v.Int()
		case semconv25.AttributeNetPeerName:
			if s.serverAddress == "" {
				// net.peer.name is deprecated, so has lower priority
				// only set when not already set with server.address
				// and allowed to be overridden by server.address.
				s.serverAddress = v.Str()
			}
		case semconv25.AttributeNetPeerPort:
			if s.serverPort == 0 {
				// net.peer.port is deprecated, so has lower priority
				// only set when not already set with server.port and
				// allowed to be overridden by server.port.
				s.serverPort = v.Int()
			}
		case semconv25.AttributeMessagingDestinationName:
			s.isMessaging = true
			s.messagingDestinationName = v.Str()
		case semconv25.AttributeMessagingOperation:
			s.isMessaging = true
		case semconv25.AttributeMessagingSystem:
			s.isMessaging = true
			s.messagingSystem = v.Str()
		case semconv25.AttributeMessagingDestinationTemporary:
			s.isMessaging = true
			s.messagingDestinationTemp = true
		case semconv25.AttributeHTTPStatusCode,
			semconv25.AttributeHTTPResponseStatusCode:
			s.isHTTP = true
			s.httpStatusCode = v.Int()
		case semconv25.AttributeHTTPMethod,
			semconv25.AttributeHTTPRequestMethod,
			semconv25.AttributeHTTPTarget,
			semconv25.AttributeHTTPScheme,
			semconv25.AttributeHTTPFlavor,
			semconv25.AttributeNetHostName:
			s.isHTTP = true
		case semconv25.AttributeURLFull,
			semconv25.AttributeHTTPURL:
			s.isHTTP = true
			// ignoring error as if parse fails then we don't want the url anyway
			s.urlFull, _ = url.Parse(v.Str())
		case semconv25.AttributeURLScheme:
			s.isHTTP = true
			s.urlScheme = v.Str()
		case semconv25.AttributeURLDomain:
			s.isHTTP = true
			s.urlDomain = v.Str()
		case semconv25.AttributeURLPort:
			s.isHTTP = true
			s.urlPort = v.Int()
		case semconv25.AttributeURLPath:
			s.isHTTP = true
			s.urlPath = v.Str()
		case semconv25.AttributeURLQuery:
			s.isHTTP = true
			s.urlQuery = v.Str()
		case semconv25.AttributeRPCGRPCStatusCode:
			s.isRPC = true
			s.grpcStatus = codes.Code(v.Int()).String()
		case semconv25.AttributeRPCSystem:
			s.isRPC = true
			s.rpcSystem = v.Str()
		case semconv25.AttributeRPCService:
			s.isRPC = true
			s.rpcService = v.Str()
		case semconv25.AttributeDBStatement,
			semconv25.AttributeDBUser:
			s.isDB = true
		case semconv25.AttributeDBName:
			s.isDB = true
			s.dbName = v.Str()
		case semconv25.AttributeDBSystem:
			s.isDB = true
			s.dbSystem = v.Str()
		case semconv27.AttributeGenAiSystem:
			s.isGenAi = true
			s.genAiSystem = v.Str()
		case semconv27.AttributeUserAgentOriginal:
			s.userAgentOriginal = v.Str()
		case semconv27.AttributeUserAgentName:
			s.userAgentName = v.Str()
		case semconv27.AttributeUserAgentVersion:
			s.userAgentVersion = v.Str()
		}
		return true
	})

	s.normalizeAttributes(userAgentParser)
	s.isTransaction = isElasticTransaction(span)
	s.enrich(span, cfg)

	spanEvents := span.Events()
	for i := 0; i < spanEvents.Len(); i++ {
		var c spanEventEnrichmentContext
		c.enrich(s, spanEvents.At(i), cfg.SpanEvent)
	}
}

func (s *spanEnrichmentContext) enrich(span ptrace.Span, cfg config.Config) {

	// In OTel, a local root span can represent an outgoing call or a producer span.
	// In such cases, the span is still mapped into a transaction, but enriched
	// with additional attributes that are specific to the outgoing call or producer span.
	isExitRootSpan := s.isTransaction && span.Kind() == ptrace.SpanKindClient || span.Kind() == ptrace.SpanKindProducer

	if s.isTransaction {
		s.enrichTransaction(span, cfg.Transaction)
	}
	if !s.isTransaction || isExitRootSpan {
		s.enrichSpan(span, cfg.Span, cfg.Transaction.Type.Enabled, isExitRootSpan)
	}
}

func (s *spanEnrichmentContext) enrichTransaction(
	span ptrace.Span,
	cfg config.ElasticTransactionConfig,
) {
	if cfg.TimestampUs.Enabled {
		span.Attributes().PutInt(elasticattr.TimestampUs, getTimestampUs(span.StartTimestamp()))
	}
	if cfg.Sampled.Enabled {
		span.Attributes().PutBool(elasticattr.TransactionSampled, s.getSampled())
	}
	if cfg.ID.Enabled {
		span.Attributes().PutStr(elasticattr.TransactionID, span.SpanID().String())
	}
	if cfg.Root.Enabled {
		span.Attributes().PutBool(elasticattr.TransactionRoot, isTraceRoot(span))
	}
	if cfg.Name.Enabled {
		span.Attributes().PutStr(elasticattr.TransactionName, span.Name())
	}
	if cfg.ProcessorEvent.Enabled {
		span.Attributes().PutStr(elasticattr.ProcessorEvent, "transaction")
	}
	if cfg.RepresentativeCount.Enabled {
		repCount := getRepresentativeCount(span.TraceState().AsRaw())
		span.Attributes().PutDouble(elasticattr.TransactionRepresentativeCount, repCount)
	}
	if cfg.DurationUs.Enabled {
		span.Attributes().PutInt(elasticattr.TransactionDurationUs, getDurationUs(span))
	}
	if cfg.Type.Enabled {
		span.Attributes().PutStr(elasticattr.TransactionType, s.getTxnType())
	}
	if cfg.Result.Enabled {
		s.setTxnResult(span)
	}
	if cfg.EventOutcome.Enabled {
		s.setEventOutcome(span)
	}
	if cfg.InferredSpans.Enabled {
		s.setInferredSpans(span)
	}
	if cfg.UserAgent.Enabled {
		s.setUserAgentIfRequired(span)
	}
}

func (s *spanEnrichmentContext) enrichSpan(
	span ptrace.Span,
	cfg config.ElasticSpanConfig,
	transactionTypeEnabled bool,
	isExitRootSpan bool,
) {
	var spanType, spanSubtype string

	if cfg.TimestampUs.Enabled {
		span.Attributes().PutInt(elasticattr.TimestampUs, getTimestampUs(span.StartTimestamp()))
	}
	if cfg.Name.Enabled {
		span.Attributes().PutStr(elasticattr.SpanName, span.Name())
	}
	if cfg.RepresentativeCount.Enabled {
		repCount := getRepresentativeCount(span.TraceState().AsRaw())
		span.Attributes().PutDouble(elasticattr.SpanRepresentativeCount, repCount)
	}
	if cfg.TypeSubtype.Enabled {
		spanType, spanSubtype = s.setSpanTypeSubtype(span)
	}
	if cfg.EventOutcome.Enabled {
		s.setEventOutcome(span)
	}
	if cfg.DurationUs.Enabled {
		span.Attributes().PutInt(elasticattr.SpanDurationUs, getDurationUs(span))
	}
	if cfg.ServiceTarget.Enabled {
		s.setServiceTarget(span)
	}
	if cfg.DestinationService.Enabled {
		s.setDestinationService(span)
	}
	if cfg.InferredSpans.Enabled {
		s.setInferredSpans(span)
	}
	if cfg.ProcessorEvent.Enabled && !isExitRootSpan {
		span.Attributes().PutStr(elasticattr.ProcessorEvent, "span")
	}
	if cfg.UserAgent.Enabled {
		s.setUserAgentIfRequired(span)
	}

	if isExitRootSpan && transactionTypeEnabled {
		if spanType != "" {
			transactionType := spanType
			if spanSubtype != "" {
				transactionType += "." + spanSubtype
			}
			span.Attributes().PutStr(elasticattr.TransactionType, transactionType)
		}
	}
}

// normalizeAttributes sets any dependent attributes that
// might not have been explicitly set as an attribute.
func (s *spanEnrichmentContext) normalizeAttributes(userAgentPraser *uaparser.Parser) {
	if s.rpcSystem == "" && s.grpcStatus != "" {
		s.rpcSystem = "grpc"
	}
	if s.userAgentOriginal != "" && userAgentPraser != nil {
		ua := userAgentPraser.ParseUserAgent(s.userAgentOriginal)
		s.inferredUserAgentName = ua.Family
		s.inferredUserAgentVersion = ua.ToVersionString()
	}
}

func (s *spanEnrichmentContext) getSampled() bool {
	// Assumes that the method is called only for transaction
	return true
}

func (s *spanEnrichmentContext) getTxnType() string {
	txnType := "unknown"
	switch {
	case s.isMessaging:
		txnType = "messaging"
	case s.isRPC, s.isHTTP:
		txnType = "request"
	}
	return txnType
}

func (s *spanEnrichmentContext) setTxnResult(span ptrace.Span) {
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

	span.Attributes().PutStr(elasticattr.TransactionResult, result)
}

func (s *spanEnrichmentContext) setEventOutcome(span ptrace.Span) {
	// default to success outcome
	outcome := "success"
	successCount := getRepresentativeCount(span.TraceState().AsRaw())
	switch {
	case s.spanStatusCode == ptrace.StatusCodeError:
		outcome = "failure"
		successCount = 0
	case s.spanStatusCode == ptrace.StatusCodeOk:
		// keep the default success outcome
	case s.httpStatusCode >= http.StatusInternalServerError:
		// TODO (lahsivjar): Handle GRPC status code? - not handled in apm-data
		// TODO (lahsivjar): Move to HTTPResponseStatusCode? Backward compatibility?
		outcome = "failure"
		successCount = 0
	}
	span.Attributes().PutStr(elasticattr.EventOutcome, outcome)
	span.Attributes().PutInt(elasticattr.SuccessCount, int64(successCount))
}

func (s *spanEnrichmentContext) setSpanTypeSubtype(span ptrace.Span) (spanType string, spanSubtype string) {
	switch {
	case s.isDB:
		spanType = "db"
		spanSubtype = s.dbSystem
	case s.isMessaging:
		spanType = "messaging"
		spanSubtype = s.messagingSystem
	case s.isRPC:
		spanType = "external"
		spanSubtype = s.rpcSystem
	case s.isHTTP:
		spanType = "external"
		spanSubtype = "http"
	case s.isGenAi:
		spanType = "genai"
		spanSubtype = s.genAiSystem
	default:
		switch span.Kind() {
		case ptrace.SpanKindInternal:
			spanType = "app"
			spanSubtype = "internal"
		default:
			spanType = "unknown"
		}
	}

	span.Attributes().PutStr(elasticattr.SpanType, spanType)
	if spanSubtype != "" {
		span.Attributes().PutStr(elasticattr.SpanSubtype, spanSubtype)
	}

	return spanType, spanSubtype
}

func (s *spanEnrichmentContext) setServiceTarget(span ptrace.Span) {
	var targetType, targetName string

	if s.peerService != "" {
		targetName = s.peerService
	}

	switch {
	case s.isDB:
		targetType = "db"
		if s.dbSystem != "" {
			targetType = s.dbSystem
		}
		if s.dbName != "" {
			targetName = s.dbName
		}
	case s.isMessaging:
		targetType = "messaging"
		if s.messagingSystem != "" {
			targetType = s.messagingSystem
		}
		if !s.messagingDestinationTemp && s.messagingDestinationName != "" {
			targetName = s.messagingDestinationName
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
		if resource := getHostPort(
			s.urlFull, s.urlDomain, s.urlPort,
			s.serverAddress, s.serverPort, // fallback
		); resource != "" {
			targetName = resource
		}
	}

	if targetType != "" || targetName != "" {
		span.Attributes().PutStr(elasticattr.ServiceTargetType, targetType)
		span.Attributes().PutStr(elasticattr.ServiceTargetName, targetName)
	}
}

func (s *spanEnrichmentContext) setDestinationService(span ptrace.Span) {
	var destnResource string
	if s.peerService != "" {
		destnResource = s.peerService
	}

	switch {
	case s.isDB:
		if destnResource == "" && s.dbSystem != "" {
			destnResource = s.dbSystem
		}
	case s.isMessaging:
		if destnResource == "" && s.messagingSystem != "" {
			destnResource = s.messagingSystem
		}
		// For parity with apm-data, destn resource does not handle
		// temporary destination flag. However, it is handled by
		// service.target fields and we might want to do the same here.
		if destnResource != "" && s.messagingDestinationName != "" {
			destnResource += "/" + s.messagingDestinationName
		}
	case s.isRPC, s.isHTTP:
		if destnResource == "" {
			if res := getHostPort(
				s.urlFull, s.urlDomain, s.urlPort,
				s.serverAddress, s.serverPort, // fallback
			); res != "" {
				destnResource = res
			} else {
				// fallback to RPC service
				destnResource = s.rpcService
			}
		}
	}

	if destnResource != "" {
		span.Attributes().PutStr(elasticattr.SpanDestinationServiceResource, destnResource)
	}
}

func (s *spanEnrichmentContext) setInferredSpans(span ptrace.Span) {
	spanLinks := span.Links()
	childIDs := pcommon.NewSlice()
	spanLinks.RemoveIf(func(spanLink ptrace.SpanLink) (remove bool) {
		spanID := spanLink.SpanID()
		spanLink.Attributes().Range(func(k string, v pcommon.Value) bool {
			switch k {
			case "is_child", "elastic.is_child":
				if v.Bool() && !spanID.IsEmpty() {
					remove = true // remove the span link if it has the child attrs
					childIDs.AppendEmpty().SetStr(hex.EncodeToString(spanID[:]))
				}
				return false // stop the loop
			}
			return true
		})
		return remove
	})

	if childIDs.Len() > 0 {
		childIDs.MoveAndAppendTo(span.Attributes().PutEmptySlice(elasticattr.ChildIDs))
	}
}

func (s *spanEnrichmentContext) setUserAgentIfRequired(span ptrace.Span) {
	if s.userAgentName == "" && s.inferredUserAgentName != "" {
		span.Attributes().PutStr(semconv27.AttributeUserAgentName, s.inferredUserAgentName)
	}
	if s.userAgentVersion == "" && s.inferredUserAgentVersion != "" {
		span.Attributes().PutStr(semconv27.AttributeUserAgentVersion, s.inferredUserAgentVersion)
	}
}

type spanEventEnrichmentContext struct {
	exceptionType    string
	exceptionMessage string

	exception        bool
	exceptionEscaped bool
}

func (s *spanEventEnrichmentContext) enrich(
	parentCtx *spanEnrichmentContext,
	se ptrace.SpanEvent,
	cfg config.SpanEventConfig,
) {
	// Extract top level span event information.
	s.exception = se.Name() == "exception"
	if s.exception {
		se.Attributes().Range(func(k string, v pcommon.Value) bool {
			switch k {
			case semconv25.AttributeExceptionEscaped:
				s.exceptionEscaped = v.Bool()
			case semconv25.AttributeExceptionType:
				s.exceptionType = v.Str()
			case semconv25.AttributeExceptionMessage:
				s.exceptionMessage = v.Str()
			}
			return true
		})
	}

	// Enrich span event attributes.
	if cfg.TimestampUs.Enabled {
		se.Attributes().PutInt(elasticattr.TimestampUs, getTimestampUs(se.Timestamp()))
	}
	if cfg.ProcessorEvent.Enabled && s.exception {
		se.Attributes().PutStr(elasticattr.ProcessorEvent, "error")
	}
	if s.exceptionType == "" && s.exceptionMessage == "" {
		// Span event does not represent an exception
		return
	}

	// Span event represents exception
	if cfg.ErrorID.Enabled {
		if id, err := newUniqueID(); err == nil {
			se.Attributes().PutStr(elasticattr.ErrorID, id)
		}
	}
	if cfg.ErrorExceptionHandled.Enabled {
		se.Attributes().PutBool(elasticattr.ErrorExceptionHandled, !s.exceptionEscaped)
	}
	if cfg.ErrorGroupingKey.Enabled {
		// See https://github.com/elastic/apm-data/issues/299
		hash := md5.New()
		// ignoring errors in hashing
		if s.exceptionType != "" {
			io.WriteString(hash, s.exceptionType)
		} else if s.exceptionMessage != "" {
			io.WriteString(hash, s.exceptionMessage)
		}
		se.Attributes().PutStr(elasticattr.ErrorGroupingKey, hex.EncodeToString(hash.Sum(nil)))
	}
	if cfg.ErrorGroupingName.Enabled {
		if s.exceptionMessage != "" {
			se.Attributes().PutStr(elasticattr.ErrorGroupingName, s.exceptionMessage)
		}
	}

	// Transaction type and sampled are added as span event enrichment only for errors
	if parentCtx.isTransaction && s.exception {
		if cfg.TransactionSampled.Enabled {
			se.Attributes().PutBool(elasticattr.TransactionSampled, parentCtx.getSampled())
		}
		if cfg.TransactionType.Enabled {
			se.Attributes().PutStr(elasticattr.TransactionType, parentCtx.getTxnType())
		}
	}
}

// getRepresentativeCount returns the number of spans represented by an
// individually sampled span as per the passed tracestate header.
//
// Representative count is similar to the OTel adjusted count definition
// with a difference that representative count can also include
// dynamically calculated representivity for non-probabilistic sampling.
// In addition, the representative count defaults to 1 if the adjusted
// count is UNKNOWN or the p-value is invalid.
//
// Def: https://opentelemetry.io/docs/specs/otel/trace/tracestate-probability-sampling/#adjusted-count)
//
// The count is calculated by using p-value:
// https://opentelemetry.io/docs/reference/specification/trace/tracestate-probability-sampling/#p-value
func getRepresentativeCount(tracestate string) float64 {
	var p uint64
	otValue := getValueForKeyInString(tracestate, "ot", ',', '=')
	if otValue != "" {
		pValue := getValueForKeyInString(otValue, "p", ';', ':')

		if pValue != "" {
			p, _ = strconv.ParseUint(pValue, 10, 6)
		}
	}

	if p == 63 {
		// p-value == 63 represents zero adjusted count
		return 0.0
	}
	return math.Pow(2, float64(p))
}

func getDurationUs(span ptrace.Span) int64 {
	return int64(span.EndTimestamp()-span.StartTimestamp()) / 1000
}

func isTraceRoot(span ptrace.Span) bool {
	return span.ParentSpanID().IsEmpty()
}

func isElasticTransaction(span ptrace.Span) bool {
	flags := tracepb.SpanFlags(span.Flags())
	switch {
	case isTraceRoot(span):
		return true
	case (flags & tracepb.SpanFlags_SPAN_FLAGS_CONTEXT_HAS_IS_REMOTE_MASK) == 0:
		// span parent is unknown, fall back to span kind
		return span.Kind() == ptrace.SpanKindServer || span.Kind() == ptrace.SpanKindConsumer
	case (flags & tracepb.SpanFlags_SPAN_FLAGS_CONTEXT_IS_REMOTE_MASK) != 0:
		// span parent is remote
		return true
	}
	return false
}

// parses string format `<key>=val<seperator>`
func getValueForKeyInString(str string, key string, separator rune, assignChar rune) string {
	for {
		str = strings.TrimSpace(str)
		if str == "" {
			break
		}
		kv := str
		if sepIdx := strings.IndexRune(str, separator); sepIdx != -1 {
			kv = strings.TrimSpace(str[:sepIdx])
			str = str[sepIdx+1:]
		} else {
			str = ""
		}
		equal := strings.IndexRune(kv, assignChar)
		if equal != -1 && kv[:equal] == key {
			return kv[equal+1:]
		}
	}

	return ""
}

func getHostPort(
	urlFull *url.URL, urlDomain string, urlPort int64,
	fallbackServerAddress string, fallbackServerPort int64,
) string {
	switch {
	case urlFull != nil:
		return urlFull.Host
	case urlDomain != "":
		if urlPort == 0 {
			return urlDomain
		}
		return net.JoinHostPort(urlDomain, strconv.FormatInt(urlPort, 10))
	case fallbackServerAddress != "":
		if fallbackServerPort == 0 {
			return fallbackServerAddress
		}
		return net.JoinHostPort(fallbackServerAddress, strconv.FormatInt(fallbackServerPort, 10))
	}
	return ""
}

func getTimestampUs(ts pcommon.Timestamp) int64 {
	return int64(ts) / 1000
}

var standardStatusCodeResults = [...]string{
	"HTTP 1xx",
	"HTTP 2xx",
	"HTTP 3xx",
	"HTTP 4xx",
	"HTTP 5xx",
}

func newUniqueID() (string, error) {
	var u [16]byte
	if _, err := io.ReadFull(rand.Reader, u[:]); err != nil {
		return "", err
	}

	// convert to string
	buf := make([]byte, 32)
	hex.Encode(buf, u[:])

	return string(buf), nil
}
