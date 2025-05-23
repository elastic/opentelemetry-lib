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

package elasticattr

import "go.opentelemetry.io/collector/pdata/pcommon"

const (
	// resource s
	AgentName    = "agent.name"
	AgentVersion = "agent.version"

	// scope s
	ServiceFrameworkName    = "service.framework.name"
	ServiceFrameworkVersion = "service.framework.version"

	// span s
	TimestampUs                    = "timestamp.us"
	ProcessorEvent                 = "processor.event"
	TransactionSampled             = "transaction.sampled"
	TransactionID                  = "transaction.id"
	TransactionRoot                = "transaction.root"
	TransactionName                = "transaction.name"
	TransactionType                = "transaction.type"
	TransactionDurationUs          = "transaction.duration.us"
	TransactionResult              = "transaction.result"
	TransactionRepresentativeCount = "transaction.representative_count"
	SpanName                       = "span.name"
	SpanType                       = "span.type"
	SpanSubtype                    = "span.subtype"
	EventOutcome                   = "event.outcome"
	EventKind                      = "event.kind"
	SuccessCount                   = "event.success_count"
	ServiceTargetType              = "service.target.type"
	ServiceTargetName              = "service.target.name"
	SpanDestinationServiceResource = "span.destination.service.resource"
	SpanDurationUs                 = "span.duration.us"
	SpanRepresentativeCount        = "span.representative_count"
	ChildIDs                       = "child.id"

	// span event s
	ParentID              = "parent.id"
	ErrorID               = "error.id"
	ErrorExceptionHandled = "error.exception.handled"
	ErrorGroupingKey      = "error.grouping_key"
	ErrorGroupingName     = "error.grouping_name"
	ErrorType             = "error.type"
)

func GetTimestampUs(ts pcommon.Timestamp) int64 {
	return int64(ts) / 1000
}
