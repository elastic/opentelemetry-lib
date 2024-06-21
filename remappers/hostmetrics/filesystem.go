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
type DeviceMetrics struct {
	TotalUsage      int64
	UsedBytes       int64
	TotalInodeUsage int64
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
	deviceMetrics := make(map[deviceKey]*DeviceMetrics)
	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		switch metric.Name() {
		case "system.filesystem.usage", "system.filesystem.inodes.usage":
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				value := dp.IntValue()
				timestamp = dp.Timestamp()
				device, mpoint, fstype, ok := getAttributes(dp)
				if !ok {
					continue
				}
				// Create a unique key for each device
				key := deviceKey{device: device.Str(), mpoint: mpoint.Str(), fstype: fstype.Str()}
				if _, exists := deviceMetrics[key]; !exists {
					deviceMetrics[key] = &DeviceMetrics{}
				}
				dmetrics := deviceMetrics[key]
				if state, ok := dp.Attributes().Get("state"); ok {
					switch state.Str() {
					case "used":
						if metric.Name() == "system.filesystem.usage" {
							dmetrics.TotalUsage += value
							dmetrics.UsedBytes += value
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.used.bytes", device.Str(), mpoint.Str(), fstype.Str(), value)
						} else {
							dmetrics.TotalInodeUsage += value
						}
					case "free":
						if metric.Name() == "system.filesystem.usage" {
							dmetrics.TotalUsage += value
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.free", device.Str(), mpoint.Str(), fstype.Str(), value)
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.available", device.Str(), mpoint.Str(), fstype.Str(), value)
						} else {
							dmetrics.TotalInodeUsage += value
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.free_files", device.Str(), mpoint.Str(), fstype.Str(), value)
						}
					}
				}
			}

		}
	}

	for key, dmetrics := range deviceMetrics {
		device, mpoint, fstype := key.device, key.mpoint, key.fstype
		if dmetrics.TotalUsage > 0 {
			addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.total", device, mpoint, fstype, dmetrics.TotalUsage)
			usedPercentage := float64(dmetrics.UsedBytes) / float64(dmetrics.TotalUsage)
			addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.used.pct", device, mpoint, fstype, usedPercentage)
		}
		if dmetrics.TotalInodeUsage > 0 {
			addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.files", device, mpoint, fstype, dmetrics.TotalInodeUsage)
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

func getAttributes(dp pmetric.NumberDataPoint) (device, mpoint, fstype pcommon.Value, ok bool) {
	device, ok = dp.Attributes().Get("device")
	if !ok {
		return
	}
	mpoint, ok = dp.Attributes().Get("mountpoint")
	if !ok {
		return
	}
	fstype, ok = dp.Attributes().Get("type")

	return
}
