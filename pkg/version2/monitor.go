package version2

// 1. 初始化调度策略模块，从调度器接口获取到集群的详细信息
// 2. 测试每个集群的prometheus是否能成功获取到需要的metric，并定期收集
// 3. 测试每个集群的Job、Namespace等信息能能否成功获取到，（并缓存？）TODO:这部分功能先不开发，先完成静态的调度
// 4. 设计接口接受调度器的作业提交，解析yaml文件，并缓存作业 TODO: 完成了本地的收集测试，等待对接接口

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

// 根据指定的url地址获取到调度器发送的json文件,格式如 example.json TODO:等调度器接口写好了再做
func getJson(url string) (body []byte) {
	return getJsonWithFile(url)
}

func getJsonWithFile(fileName string) (content []byte) {
	// 打开文件
	file, err := os.Open(fileName)
	if err != nil {
		log.Println("ERROR: opening file:", err)
		return
	}
	defer file.Close()

	// 读取文件内容
	content, err = io.ReadAll(file)
	if err != nil {
		log.Println("ERROR: reading file:", err)
		return
	}
	return content
}

// 解析json
func (monitor *Monitor) unmarshalJson(body []byte) {
	err := json.Unmarshal(body, &monitor)
	if err != nil {
		log.Println("ERROR:", err)
		return
	}
	log.Println("INFO: Information initiated")
	// fmt.Printf("%+v",monitor)
}

// 根据nodename生成prommetric查询语句
func generatePromMetrics(nodeName string, nodeType string) map[string]string {
	metrics := make(map[string]string)
	cpuMetricExpr := `sum(increase(node_cpu_seconds_total{mode!="idle",node="` + nodeName + `"}[2m]))` +
		` / sum(increase(node_cpu_seconds_total{node="` + nodeName + `"}[2m]))`
	metrics["CPU_USAGE"] = cpuMetricExpr
	// 添加其他 Prometheus 指标
	metrics["TOTAL_MEMORY"] = `node_memory_MemTotal_bytes{node="` + nodeName + `"}`    // 内存总量
	metrics["FREE_MEMORY"] = `node_memory_MemAvailable_bytes{node="` + nodeName + `"}` // 可用内存
	if nodeType == "GPU" {
		metrics["GPU_UTIL"] = `DCGM_FI_DEV_GPU_UTIL{Hostname="` + nodeName + `"}`
		metrics["GPU_MEMORY_FREE"] = `DCGM_FI_DEV_FB_FREE{Hostname="` + nodeName + `"}`
		metrics["GPU_MEMORY_USED"] = `DCGM_FI_DEV_FB_USED{Hostname="` + nodeName + `"}`
	}
	return metrics
}

func (monitor *Monitor) getMetric() {
	for dc, datacenter := range monitor.DataCenterInfo {
		for cl, cluster := range datacenter.ClusterInfo {
			for n, node := range cluster.NodeInfo {

				// 异常处理
				defer func() {
					if r := recover(); r != nil {
						log.Printf("ERROR: GetMetric error! DatacenterID:%v ClusterID:%v NodeID:%v", datacenter.DataCenterID, cluster.ClusterID, node.NodeID)
					}
				}()

				nodeMetric := make(map[string]Data)
				metrics := generatePromMetrics(node.NodeID, node.NodeType)
				// 以node为单位获取metric
				for metric, metricExpr := range metrics {
					var promResponse PromResponse
					metricExpr = url.QueryEscape(metricExpr)
					prometheusURL := "http://" + cluster.ClusterPromIpPort + "/api/v1/query?query=" + metricExpr
					resp, err := http.Get(prometheusURL)
					if err != nil {
						log.Println("ERROR: sending prometheus request failed", err)
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
				if node.TOTAL_MEMORY == 0 {
					log.Printf("ERROR: invalid metric at DCID:%v ClID:%v NID:%v", monitor.DataCenterInfo[dc].DataCenterID, monitor.DataCenterInfo[dc].ClusterInfo[cl].ClusterID, monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n].NodeID)
				}
				node.FREE_MEMORY, _ = strconv.ParseInt(nodeMetric["FREE_MEMORY"].Result[0].Value[1].(string), 10, 64)
				if node.NodeType == "GPU" {
					for _, result := range nodeMetric["GPU_UTIL"].Result {
						node.FindCard(result.Metric["gpu"].(string)).GPU_UTIL, _ = strconv.ParseInt(result.Value[1].(string), 10, 64)
					}
					for _, result := range nodeMetric["GPU_MEMORY_FREE"].Result {
						node.FindCard(result.Metric["gpu"].(string)).GPU_MEMORY_FREE, _ = strconv.ParseInt(result.Value[1].(string), 10, 64)
					}
					for _, result := range nodeMetric["GPU_MEMORY_USED"].Result {
						node.FindCard(result.Metric["gpu"].(string)).GPU_MEMORY_USED, _ = strconv.ParseInt(result.Value[1].(string), 10, 64)
					}
				}
			}
		}
	}
	// fmt.Printf("%+v", *monitor)
	log.Println("INFO: GetMetric done!")
}

// 创建集群外的客户端
func NewClientSetOutOfCluster(kubeconfig string) (client *kubernetes.Clientset, err error) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	// log.Println("clientset succeed !")
	return clientset, nil
}

func (monitor *Monitor) NewClientSetForEachCluseter() {
	for _, datacenter := range monitor.DataCenterInfo {
		for _, cluster := range datacenter.ClusterInfo {
			var err error
			cluster.ClusterClientSet, err = NewClientSetOutOfCluster(cluster.ClusterKubeconfigFilePath)
			if err != nil {
				log.Println("ERROR: NewClientSetForEachCluseter failed!\t", err)
			}
		}
	}
}

// TODO: 获取集群特定namespace的Job信息 还没测试
func jobList(client *kubernetes.Clientset, namespace string) (joblist *batchv1.JobList, err error) {
	joblist, err = client.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Println("ERROR: cannot list jobs", err)
		return nil, err
	}
	return joblist, nil
}

func parseYamlFile(filePath string) (batchv1.Job, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return batchv1.Job{}, fmt.Errorf("failed to read file %s: %v", filePath, err)
	}

	// 使用 Kubernetes 库解析 YAML
	// 这里假设 YAML 文件是一个 Pod 对象
	var job batchv1.Job
	err = yaml.Unmarshal(data, &job)
	if err != nil {
		return batchv1.Job{}, fmt.Errorf("failed to unmarshal YAML from file %s: %v", filePath, err)
	}

	// 打印出解析出来的名称作为示例
	// log.Printf("INFO: Parsed Job: %s\n", job.Name)
	return job, nil
}

// TODO: 从调度器接口获取Job信息（起个http服务什么的）
func (monitor *Monitor) getJob() {
	monitor.getJobWithFile(`yaml_template`)
	log.Println("INFO: getJob finished")
}

func (monitor *Monitor) getJobWithFile(directory string) {
	dirEntry, err := os.ReadDir(directory)
	if err != nil {
		log.Println("ERROR: getJobWithFile failed!", err)
	}
	for _, file := range dirEntry { // TODO: 可以考虑开发定期查看目录并更新的功能
		if filepath.Ext(file.Name()) == ".yaml" || filepath.Ext(file.Name()) == ".yml" {
			filePath := filepath.Join(directory, file.Name())
			jobSpec, err := parseYamlFile(filePath)
			if err != nil {
				log.Println("ERROR: process file failed", err)
			}
			JobModelName := jobSpec.Annotations[`model_name`]
			JobDataSize, _ := strconv.ParseInt(jobSpec.Annotations[`data_size`], 10, 64)
			JobEpoch, _ := strconv.ParseInt(jobSpec.Annotations[`epoch`], 10, 64)
			monitor.JobPool.OriginJobQueue = append(monitor.JobPool.OriginJobQueue, &Job{JobSpec: jobSpec, YamlFilePath: filePath, JobModelName: JobModelName, DataSize: JobDataSize, Epoch: JobEpoch})
		}
	}
	// fmt.Printf("%+v", monitor)
}

// Monitor的整体工作逻辑
func NewMonitor() *Monitor {
	monitor := &Monitor{}

	// 从接口读取基础信息，初始化数据结构 TODO: 正式版需要修改读取Json的方式
	monitor.unmarshalJson(getJson("example2.json"))

	// 为每个集群生成一个clientset
	monitor.NewClientSetForEachCluseter()

	// 每隔一分钟更新一次metric
	// go func() {
	// 	for {
	// 		time.Sleep(time.Minute)
	// 		monitor.getMetric()
	// 	}
	// }()
	monitor.getMetric()

	// 获取Job
	monitor.getJob()

	return monitor
}

func AssignJobWithSystem(job *Job) bool { // TODO:通过调度器后台来分发作业，由浪潮完成
	return true
}

func (monitor *Monitor) AssignJob() {
	failedJobQueue := []*Job{}
	for _, job := range monitor.JobPool.AssignedJob {
		if AssignJobWithSystem(job) {
			monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob, job)
		} else {
			failedJobQueue = append(failedJobQueue, job)
		}
	}
	monitor.JobPool.ScheduledJob = failedJobQueue
}
