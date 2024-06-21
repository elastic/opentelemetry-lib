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

package hostmetrics

import (
	remappers "github.com/elastic/opentelemetry-lib/remappers/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// DeviceMetrics holds metrics data for a device
type deviceMetrics struct {
	totalUsage      int64
	usedBytes       int64
	totalInodeUsage int64
}

type deviceKey struct {
	device string
	mpoint string
	fstype string
}

type number interface {
	~int64 | ~float64
}

func remapFilesystemMetrics(src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
) error {
	var timestamp pcommon.Timestamp
	deviceMetricsMap := make(map[deviceKey]*deviceMetrics)
	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		switch metric.Name() {
		case "system.filesystem.usage", "system.filesystem.inodes.usage":
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				value := dp.IntValue()
				timestamp = dp.Timestamp()
				key, ok := getDeviceKey(dp)
				if !ok {
					continue
				}
				if _, exists := deviceMetricsMap[key]; !exists {
					deviceMetricsMap[key] = &deviceMetrics{}
				}
				dmetrics := deviceMetricsMap[key]
				if state, ok := dp.Attributes().Get("state"); ok {
					switch state.Str() {
					case "used":
						if metric.Name() == "system.filesystem.usage" {
							dmetrics.totalUsage += value
							dmetrics.usedBytes += value
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.used.bytes", key.device, key.mpoint, key.fstype, value)
						} else {
							dmetrics.totalInodeUsage += value
						}
					case "free":
						if metric.Name() == "system.filesystem.usage" {
							dmetrics.totalUsage += value
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.free", key.device, key.mpoint, key.fstype, value)
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.available", key.device, key.mpoint, key.fstype, value)
						} else {
							dmetrics.totalInodeUsage += value
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.free_files", key.device, key.mpoint, key.fstype, value)
						}
					}
				}
			}

		}
	}

	for key, dmetrics := range deviceMetricsMap {
		device, mpoint, fstype := key.device, key.mpoint, key.fstype
		if dmetrics.totalUsage > 0 {
			addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.total", device, mpoint, fstype, dmetrics.totalUsage)
			usedPercentage := float64(dmetrics.usedBytes) / float64(dmetrics.totalUsage)
			addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.used.pct", device, mpoint, fstype, usedPercentage)
		}
		if dmetrics.totalInodeUsage > 0 {
			addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.files", device, mpoint, fstype, dmetrics.totalInodeUsage)
		}

	}
	return nil
}

func addFileSystemMetrics[T number](out pmetric.MetricSlice,
	timestamp pcommon.Timestamp,
	dataset, name, device, mpoint, fstype string,
	value T,
) {
	var intValue *int64
	var doubleValue *float64
	if i, ok := any(value).(int64); ok {
		intValue = &i
	} else if d, ok := any(value).(float64); ok {
		doubleValue = &d
	}

	remappers.AddMetrics(out, dataset,
		func(dp pmetric.NumberDataPoint) {
			dp.Attributes().PutStr("system.filesystem.device_name", device)
			dp.Attributes().PutStr("system.filesystem.mount_point", mpoint)
			dp.Attributes().PutStr("system.filesystem.type", fstype)
		},
		remappers.Metric{
			DataType:    pmetric.MetricTypeSum,
			Name:        name,
			Timestamp:   timestamp,
			IntValue:    intValue,
			DoubleValue: doubleValue,
		},
	)

}

func getDeviceKey(dp pmetric.NumberDataPoint) (key deviceKey, ok bool) {
	device, ok := dp.Attributes().Get("device")
	if !ok {
		return
	}
	mpoint, ok := dp.Attributes().Get("mountpoint")
	if !ok {
		return
	}
	fstype, ok := dp.Attributes().Get("type")
	if !ok {
		return
	}

	key = deviceKey{device: device.Str(), mpoint: mpoint.Str(), fstype: fstype.Str()}

	return
}
