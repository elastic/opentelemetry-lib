package hostmetrics

import (
	"strings"

	remappers "github.com/elastic/opentelemetry-lib/remappers/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"golang.org/x/exp/constraints"
)

func remapFilesystemMetrics(src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
) error {
	var timestamp pcommon.Timestamp
	var device, mpoint, fstype string
	totalUsagePerDevice := make(map[string]int64)
	totalInodeUsagePerDevice := make(map[string]int64)
	usedBytesPerDevice := make(map[string]int64)

	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		switch metric.Name() {
		case "system.filesystem.usage", "system.filesystem.inodes.usage":
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				value := dp.IntValue()
				timestamp = dp.Timestamp()
				deviceValue, mpointValue, fstypeValue, ok := getAttributes(dp)
				if !ok {
					continue
				}
				device, mpoint, fstype = deviceValue.Str(), mpointValue.Str(), fstypeValue.Str()
				// Create a unique key for each device
				deviceKey := device + "_" + mpoint + "_" + fstype
				if state, ok := dp.Attributes().Get("state"); ok {
					switch state.Str() {
					case "used":
						if metric.Name() == "system.filesystem.usage" {
							totalUsagePerDevice[deviceKey] += value
							usedBytesPerDevice[deviceKey] += value
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.used.bytes", device, mpoint, fstype, value)
						} else {
							totalInodeUsagePerDevice[deviceKey] += value
						}
					case "free":
						if metric.Name() == "system.filesystem.usage" {
							totalUsagePerDevice[deviceKey] += value
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.free", device, mpoint, fstype, value)
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.available", device, mpoint, fstype, value)
						} else {
							totalInodeUsagePerDevice[deviceKey] += value
							addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.free_files", device, mpoint, fstype, value)
						}
					}
				}
			}

		}
	}

	for deviceKey, totalfsusage := range totalUsagePerDevice {
		device, mpoint, fstype = parseDeviceKey(deviceKey)
		addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.total", device, mpoint, fstype, totalfsusage)
		if usedBytes, exists := usedBytesPerDevice[deviceKey]; exists {
			usedPercentage := float64(usedBytes) / float64(totalfsusage)
			addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.used.pct", device, mpoint, fstype, usedPercentage)
		}
	}

	for deviceKey, totalinodeusage := range totalInodeUsagePerDevice {
		device, mpoint, fstype = parseDeviceKey(deviceKey)
		addFileSystemMetrics(out, timestamp, dataset, "system.filesystem.files", device, mpoint, fstype, totalinodeusage)
	}
	return nil
}

func addFileSystemMetrics[T interface {
	constraints.Integer | constraints.Float
}](out pmetric.MetricSlice,
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

func parseDeviceKey(devicekey string) (device, mpoint, fstype string) {
	parts := strings.Split(devicekey, "_")
	if len(parts) != 3 {
		return "", "", ""
	}
	return parts[0], parts[1], parts[2]
}
