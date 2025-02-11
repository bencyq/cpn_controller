package version2

import (
	"math"
)

//  作业队列按照FIFO顺序排列
// 对于一个作业，模拟其在各个集群运行的完成时间，并进行分配

// 对于一个作业，模拟其在各个集群运行的完成时间
func (monitor *Monitor) OptimalAllocate(job *Job) bool {
	var optAlc = [4]int{} // optimal Allocation, 存储job的分配位置
	minTotalTime := int64(math.MaxInt64)

	// 模拟计算作业在各个集群的各个节点的运行时间+传输时间，选取时间最少的
	for dc, dataCenterInfo := range monitor.DataCenterInfo {
		for cl, clusterInfo := range dataCenterInfo.ClusterInfo {
			transferTime := (job.DataSize * 1024) / clusterInfo.Bandwidth
			for n, nodeInfo := range clusterInfo.NodeInfo {
				if nodeInfo.CPU_USAGE > 0.7 {
					continue
				}
				if nodeInfo.FREE_MEMORY-10 < job.MemoryReq {
					continue
				}
				for c, cardInfo := range nodeInfo.CardInfo {
					if cardInfo.GPU_MEMORY_FREE-1024 < job.GPUMemoryReq || len(cardInfo.JobQueue) >= 3 {
						continue
					}
					for _, job := range cardInfo.JobQueue {
						if job.JobType == `GPU` {
							continue
						}
					}
					runtime := monitor.RuntimePredict(job, dataCenterInfo.DataCenterID, clusterInfo.ClusterID, nodeInfo.NodeID, cardInfo.CardID)
					if runtime+transferTime < minTotalTime {
						minTotalTime = runtime + transferTime
						optAlc[0] = dc
						optAlc[1] = cl
						optAlc[2] = n
						optAlc[3] = c
					}
				}
			}
		}
	}

	// 如果没有合适的位置分配任务
	if optAlc[0] == 0 {
		return false
	}

	// 给对应的card上的jobqueue挂上作业
	monitor.DataCenterInfo[optAlc[0]].ClusterInfo[optAlc[1]].NodeInfo[optAlc[2]].CardInfo[optAlc[3]].JobQueue = append(monitor.DataCenterInfo[optAlc[0]].ClusterInfo[optAlc[1]].NodeInfo[optAlc[2]].CardInfo[optAlc[3]].JobQueue, job)

	// 在对应的Card上，减去该模型预测需要占用的资源 TODO: 目前只考虑了显存
	monitor.DataCenterInfo[optAlc[0]].ClusterInfo[optAlc[1]].NodeInfo[optAlc[2]].CardInfo[optAlc[3]].GPU_MEMORY_USED += job.GPUMemoryReq
	monitor.DataCenterInfo[optAlc[0]].ClusterInfo[optAlc[1]].NodeInfo[optAlc[2]].CardInfo[optAlc[3]].GPU_MEMORY_FREE -= job.GPUMemoryReq
	return true
}
