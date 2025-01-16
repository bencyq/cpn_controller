package version2

// 1. 初始化调度策略模块，从调度器接口获取到集群的详细信息
// 2. 测试每个集群的prometheus是否能成功获取到需要的metric，并定期收集
// 3. 测试每个集群的Job、Namespace等信息能能否成功获取到，（并缓存？）

import (
	// "fmt"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

// 根据指定的url地址获取到调度器发送的json文件,格式如 example.json TODO:等调度器接口写好了再做
func getJson(url string) (body []byte) {
	return nil
}

// 解析json
func unmarshalJson(body []byte) {
	var root Root
	err := json.Unmarshal(body, &root)
	if err != nil {
		log.Println("Error:", err)
		return
	}
	// fmt.Printf("%+v",root)
}

// 根据nodename生成prommetric查询语句
func generatePromMetrics(nodeName string) map[string]string {
	metrics := make(map[string]string)
	cpuMetricExpr := `sum(increase(node_cpu_seconds_total{mode!="idle",node="` + nodeName + `"}[2m]))` +
		` / sum(increase(node_cpu_seconds_total{node="` + nodeName + `"}[2m]))`
	metrics["CPU_USAGE"] = cpuMetricExpr
	// 添加其他 Prometheus 指标
	metrics["TOTAL_MEMORY"] = `node_memory_MemTotal_bytes{node="` + nodeName + `"}`    // 内存总量
	metrics["FREE_MEMORY"] = `node_memory_MemAvailable_bytes{node="` + nodeName + `"}` // 可用内存
	metrics["GPU_UTIL"] = `DCGM_FI_DEV_GPU_UTIL{Hostname="` + nodeName + `"}`
	metrics["GPU_MEMORY_FREE"] = `DCGM_FI_DEV_FB_FREE{Hostname="` + nodeName + `"}`
	metrics["GPU_MEMORY_USED"] = `DCGM_FI_DEV_FB_USED{Hostname="` + nodeName + `"}`
	return metrics
}

// TODO: 需要测试
func getMetric(cluster *ClusterInfo) {
	for _, node := range cluster.NodeInfo {
		metrics := generatePromMetrics(node.NodeID)
		nodeMetric := make(map[string]Data)

		// 以node为单位获取metric
		for metric, metricExpr := range metrics {
			var promResponse PromResponse
			metricExpr = url.QueryEscape(metricExpr)
			prometheusURL := "http://" + cluster.ClusterPromIpPort + "/api/v1/query?query=" + metricExpr
			resp, err := http.Get(prometheusURL)
			if err != nil {
				log.Println("Error: sending prometheus request failed", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			_ = json.Unmarshal(body, &promResponse)

			if promResponse.Status == "success" {
				nodeMetric[metric] = promResponse.Data
			} else {
				log.Println("promResponse wrong")
			}
		}

		// 解析nodeMetric并把值存入对应的变量中
		node.CPU_USAGE, _ = strconv.ParseFloat(nodeMetric["CPU_USAGE"].Result[0].Value[1].(string), 64)
		node.TOTAL_MEMORY, _ = strconv.ParseInt(nodeMetric["TOTAL_MEMORY"].Result[0].Value[1].(string), 10, 64)
		node.FREE_MEMORY, _ = strconv.ParseInt(nodeMetric["FREE_MEMORY"].Result[0].Value[1].(string), 10, 64)
		for _, result := range nodeMetric["GPU_UTIL"].Result {
			node.FindCard(result.Metric["gpu"].(string)).GPU_UTIL, _ = strconv.ParseInt(result.Value[1].(string), 10, 64)
		}
		for _, result := range nodeMetric["GPU_MEMORY_FREE"].Result {
			node.FindCard(result.Metric["gpu"].(string)).GPU_UTIL, _ = strconv.ParseInt(result.Value[1].(string), 10, 64)
		}
		for _, result := range nodeMetric["GPU_MEMORY_USED"].Result {
			node.FindCard(result.Metric["gpu"].(string)).GPU_UTIL, _ = strconv.ParseInt(result.Value[1].(string), 10, 64)
		}

	}
}
