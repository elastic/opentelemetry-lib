package hostmetrics

import (
	remappers "github.com/elastic/opentelemetry-lib/remappers/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func remapFilesystemMetrics(src, out pmetric.MetricSlice,
	_ pcommon.Resource,
	dataset string,
) error {
	var totalfsusage, totalinodeusage int64
	var timestamp pcommon.Timestamp
	var esfsname, device, mpoint, fstype string

	for i := 0; i < src.Len(); i++ {
		metric := src.At(i)
		switch metric.Name() {
		case "system.filesystem.usage":
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
				if state, ok := dp.Attributes().Get("state"); ok {
					switch state.Str() {
					case "used":
						esfsname = "system.filesystem.used.bytes"
						totalfsusage += value
						addFileSystemMetrics(out, timestamp, dataset, esfsname, device, mpoint, fstype, value)
					case "free":
						totalfsusage += value
						esfsname = "system.filesystem.free"
						addFileSystemMetrics(out, timestamp, dataset, esfsname, device, mpoint, fstype, value)
						esfsname = "system.filesystem.available"
						addFileSystemMetrics(out, timestamp, dataset, esfsname, device, mpoint, fstype, value)
					}
				}
			}
			if totalfsusage > 0 {
				esfsname = "system.filesystem.total"
				addFileSystemMetrics(out, timestamp, dataset, esfsname, device, mpoint, fstype, totalfsusage)
			}

		case "system.filesystem.inodes.usage":
			dataPoints := metric.Sum().DataPoints()
			for j := 0; j < dataPoints.Len(); j++ {
				dp := dataPoints.At(j)
				value := dp.IntValue()
				deviceValue, mpointValue, fstypeValue, ok := getAttributes(dp)
				if !ok {
					continue
				}
				device, mpoint, fstype = deviceValue.Str(), mpointValue.Str(), fstypeValue.Str()
				if state, ok := dp.Attributes().Get("state"); ok {
					switch state.Str() {
					case "used":
						totalinodeusage += value
					case "free":
						totalinodeusage += value
						esfsname = "system.filesystem.free_files"
						addFileSystemMetrics(out, dp.Timestamp(), dataset, esfsname, device, mpoint, fstype, value)
					}
				}
			}
			esfsname = "system.filesystem.files"
			addFileSystemMetrics(out, timestamp, dataset, esfsname, device, mpoint, fstype, totalinodeusage)
		}
	}
	return nil
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

func addFileSystemMetrics(out pmetric.MetricSlice,
	timestamp pcommon.Timestamp,
	dataset, name, device, mpoint, fstype string,
	value int64,
) {

	remappers.AddMetrics(out, dataset,
		func(dp pmetric.NumberDataPoint) {
			dp.Attributes().PutStr("system.filesystem.device_name", device)
			dp.Attributes().PutStr("system.filesystem.mount_point", mpoint)
			dp.Attributes().PutStr("system.filesystem.type", fstype)
		},
		remappers.Metric{
			DataType:  pmetric.MetricTypeSum,
			Name:      name,
			Timestamp: timestamp,
			IntValue:  &value,
		},
	)

}
