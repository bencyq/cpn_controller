package version2

import (
	"log"
	"path/filepath"
	"strconv"
)

// 5. 在每个集群的每台服务器上运行基准测试程序，获得评价指标（暂定resnet50、yolov8m、llama3，每个各10mins）
// 6. 实现预测器的功能（能够根据提供的模型信息，给出指标），预测其在A100上的平均运行时间

// 检测每个节点是否已经跑过benchmark
func (monitor *Monitor) checkBenchMark() {
	for _, datacenter := range monitor.DataCenterInfo {
		for _, cluster := range datacenter.ClusterInfo {
			for _, node := range cluster.NodeInfo {
				if node.BenchMark.Model1AVGRunTime == 0.0 {
					monitor.runBenchMark(datacenter.DataCenterID, cluster.ClusterID, node.NodeID)
					log.Printf("INFO: No BenchMark in DataCenter: %v\tClusterID: %v\tNodeID: %v\t", datacenter.DataCenterID, cluster.ClusterID, node.NodeID)
				}
			}
		}
	}
}

// 运行基准测试程序，获得评价指标 TODO: 先在json文件里手动配置，后续增加功能
func (monitor *Monitor) runBenchMark(DataCenterID string, ClusterID string, NodeID string) {

}

// 读取并解析model_baseline.csv文件
func (monitor *Monitor) readModelBaseline() {
	// 获取项目工作目录，并读取model_baseline.csv文件
	root, err := getProjectRoot()
	if err != nil {
		log.Println("ERROR: JobAnalyze faild", err)
	}
	fp := filepath.Join(root, "pkg", "version2", "model_baseline.csv")

	// 解析 TODO:
	_, lines := ReadCsv(fp)
	var modelBaseline = map[string][]string{}
	for _, ele := range lines {
		modelBaseline[ele[0]] = ele[1:]
	}
	monitor.ModelBaseline = modelBaseline
}

// 作业分析器 分析作业的memoryReq、JobType等数据 TODO:现在都是静态配置，之后可以设计动态配置
func (monitor *Monitor) JobAnalyze(job *Job) {
	if _, exists := monitor.ModelBaseline[job.JobModelName]; exists {
		job.GPUMemoryReq, _ = strconv.ParseInt(monitor.ModelBaseline[job.JobModelName][0], 10, 64)
	} else {
	}

	if job.JobModelName == "llama3" || job.JobModelName == "glm4" || job.JobModelName == "qwen2.5" {
		job.JobType = "GPU"
	} else {
		job.JobType = "CPU"
	}
}

// 预测器 TODO:
func (monitor *Monitor) RuntimePredict(job *Job, dataCenterID string, clusterID string, nodeID string, cardID string) (runtime int64) {
	// 分析当前该卡上有的作业
	// 分析该作业的预计运行时间
	return 300 // 以秒为单位
}

// TODO:  未测试 FIXME:
func (monitor *Monitor) InitPredictor() {
	monitor.readModelBaseline()
	var SchduleFailedJob = []*Job{}
	for _, job := range monitor.JobPool.OriginJobQueue {
		monitor.JobAnalyze(job)
		if monitor.OptimalAllocate(job) {
			monitor.JobPool.ScheduledJob = append(monitor.JobPool.ScheduledJob, job)
		} else {
			SchduleFailedJob = append(SchduleFailedJob, job)
		}
	}
	monitor.JobPool.OriginJobQueue = SchduleFailedJob
}
