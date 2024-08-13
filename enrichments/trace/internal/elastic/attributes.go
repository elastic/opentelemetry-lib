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

const (
	// resource attributes
	AttributeAgentName    = "agent.name"
	AttributeAgentVersion = "agent.version"

	// scope attributes
	AttributeServiceFrameworkName    = "service.framework.name"
	AttributeServiceFrameworkVersion = "service.framework.version"

	// span attributes
	AttributeProcessorEvent                 = "processor.event"
	AttributeTransactionID                  = "transaction.id"
	AttributeTransactionRoot                = "transaction.root"
	AttributeTransactionName                = "transaction.name"
	AttributeTransactionType                = "transaction.type"
	AttributeTransactionDurationUs          = "transaction.duration.us"
	AttributeTransactionResult              = "transaction.result"
	AttributeSpanName                       = "span.name"
	AttributeSpanType                       = "span.type"
	AttributeSpanSubtype                    = "span.subtype"
	AttributeEventOutcome                   = "event.outcome"
	AttributeSuccessCount                   = "event.success_count"
	AttributeServiceTargetType              = "service.target.type"
	AttributeServiceTargetName              = "service.target.name"
	AttributeSpanDestinationServiceResource = "span.destination.service.resource"
	AttributeSpanDurationUs                 = "span.duration.us"
)
