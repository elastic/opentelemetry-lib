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

package remappedmetric

import (
	"testing"
	"time"

	"github.com/elastic/opentelemetry-lib/remappers/internal/testutils"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestAdd(t *testing.T) {
	dataset := "test"
	now := pcommon.NewTimestampFromTime(time.Now())
	attrs := map[string]any{
		"data_stream.dataset": dataset,
		"otel_remapped":       true,
	}

	for _, tc := range []struct {
		name     string
		input    []Metric
		mutator  func(pmetric.NumberDataPoint)
		expected pmetric.MetricSlice
	}{
		{
			name:     "empty",
			input:    nil,
			mutator:  EmptyMutator,
			expected: pmetric.NewMetricSlice(),
		},
		{
			name: "no_value_set",
			input: []Metric{
				{
					Name: "invalid",
				},
				{
					// value and datatype not set
					Timestamp: now,
				},
				{
					// timestamp and datatype not set
					IntValue: testutils.Ptr(int64(10)),
				},
				{
					// timestamp and datatype not set
					DoubleValue: testutils.Ptr(10.4),
				},
				{
					// timestamp and value not set
					DataType: pmetric.MetricTypeSum,
				},
				{
					Name:      "invalid_histo",
					DataType:  pmetric.MetricTypeHistogram,
					Timestamp: now,
					IntValue:  testutils.Ptr(int64(10)),
				},
				{
					Name:      "invalid_exp_histo",
					DataType:  pmetric.MetricTypeExponentialHistogram,
					Timestamp: now,
					IntValue:  testutils.Ptr(int64(10)),
				},
				{
					Name:      "invalid_summary",
					DataType:  pmetric.MetricTypeSummary,
					Timestamp: now,
					IntValue:  testutils.Ptr(int64(10)),
				},
			},
			mutator:  EmptyMutator,
			expected: pmetric.NewMetricSlice(),
		},
		{
			name: "valid",
			input: []Metric{
				{
					Name:      "valid_int",
					DataType:  pmetric.MetricTypeSum,
					Timestamp: now,
					IntValue:  testutils.Ptr(int64(10)),
				},
				{
					Name:        "valid_double",
					DataType:    pmetric.MetricTypeSum,
					Timestamp:   now,
					DoubleValue: testutils.Ptr(10.4),
				},
			},
			mutator: EmptyMutator,
			expected: func() pmetric.MetricSlice {
				s := pmetric.NewMetricSlice()
				m := s.AppendEmpty()
				m.SetName("valid_int")
				dp := m.SetEmptySum().DataPoints().AppendEmpty()
				dp.SetIntValue(10)
				dp.SetTimestamp(now)
				dp.Attributes().FromRaw(attrs)

				m = s.AppendEmpty()
				m.SetName("valid_double")
				dp = m.SetEmptySum().DataPoints().AppendEmpty()
				dp.SetDoubleValue(10.4)
				dp.SetTimestamp(now)
				dp.Attributes().FromRaw(attrs)

				return s
			}(),
		},
	} {
		actual := pmetric.NewMetricSlice()
		Add(actual, dataset, tc.mutator, tc.input...)

		assert.Empty(t, cmp.Diff(
			testutils.MetricSliceToTestMetric(t, tc.expected),
			testutils.MetricSliceToTestMetric(t, actual),
		))
	}
}
