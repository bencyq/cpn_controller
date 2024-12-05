package cpnclient

import (
	"time"
)

const CpnServerURL = "http://10.90.1.49:23981"

const TimeInterval = 60 * time.Second

// 客户端的http服务器监听地址
const ClientIP = "0.0.0.0:23980"

// 集群名字
const ClientName = "cluster-one"

var WorkerNodeName = []string{
	`node16`,
	`node235`,
}

// 计算2分钟内的cpu使用量 FIXME: 获取1分钟以内的使用量会得到空值

// TODO:把指标按照nodename分类好，要不然调度器太难处理了

var PromMetrics = func() map[string]map[string]string {
	promMetrics := make(map[string]map[string]string)
	for _, nodename := range WorkerNodeName {
		metrics := make(map[string]string)
		cpuMetricExpr := `sum(increase(node_cpu_seconds_total{mode!="idle",node="` + nodename + `"}[2m]))` +
			` / sum(increase(node_cpu_seconds_total{node="` + nodename + `"}[2m]))`
		metrics["CPU_USAGE"] = cpuMetricExpr
		// 添加其他 Prometheus 指标
		metrics["TOTAL_MEMORY"] = `node_memory_MemTotal_bytes{node="` + nodename + `"}`    // 内存总量
		metrics["FREE_MEMORY"] = `node_memory_MemAvailable_bytes{node="` + nodename + `"}` // 可用内存
		metrics["GPU_UTIL"] = `DCGM_FI_DEV_GPU_UTIL{node="` + nodename + `"}`
		metrics["GPU_MEMORY_FREE"] = `DCGM_FI_DEV_FB_FREE{node="` + nodename + `"}`
		metrics["GPU_MEMORY_USED"] = `DCGM_FI_DEV_FB_USED{node="` + nodename + `"}`
		promMetrics[nodename] = metrics
	}
	return promMetrics
}()

// var PromMetrics = func() map[string]string {
// 	metrics := make(map[string]string)

// 	// 添加 CPU 使用率指标
// 	for _, nodename := range WorkerNodeName {
// 		cpuMetricName := fmt.Sprintf("cpu_usage_%v", nodename) // 使用节点名作为指标的键名
// 		cpuMetricExpr := `sum(increase(node_cpu_seconds_total{mode!="idle",node="` + fmt.Sprintf("%v", nodename) + `"}[2m]))` +
// 			` / sum(increase(node_cpu_seconds_total{node="` + fmt.Sprintf("%v", nodename) + `"}[2m]))`
// 		metrics[cpuMetricName] = cpuMetricExpr
// 	}

// 	// 添加其他 Prometheus 指标
// 	metrics["MEMORY_USAGE"] = `node_memory_MemTotal_bytes`    // 内存总量
// 	metrics["FREE_MEMORY"] = `node_memory_MemAvailable_bytes` // 可用内存
// 	metrics["GPU_UTIL"] = `DCGM_FI_DEV_GPU_UTIL`
// 	metrics["GPU_MEMORY_FREE"] = `DCGM_FI_DEV_FB_FREE`
// 	metrics["GPU_MEMORY_USED"] = `DCGM_FI_DEV_FB_USED`

// 	return metrics
// }()
