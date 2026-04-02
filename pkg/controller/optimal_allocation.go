package controller

import (
	"log"
	"math"
	"time"
)

//  作业队列按照FIFO顺序排列
// 对于一个作业，模拟其在各个集群运行的完成时间，并进行分配

// 对于一个作业，模拟其在各个集群运行的完成时间
func clampLoad(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func gpuMemoryUsage(used int64, free int64) (float64, bool) {
	total := used + free
	if total <= 0 {
		return 0, false
	}
	return clampLoad(float64(used) / float64(total)), true
}

func (monitor *Monitor) nodeLoad(dc int, cl int, n int) float64 {
	node := monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n]
	if node == nil {
		return 0
	}

	loadParts := []float64{clampLoad(node.CPU_USAGE)}
	if node.TOTAL_MEMORY > 0 {
		memUsage := 1 - float64(node.FREE_MEMORY)/float64(node.TOTAL_MEMORY)
		loadParts = append(loadParts, clampLoad(memUsage))
	}

	if len(node.CardInfo) > 0 {
		gpuUtilSum := 0.0
		gpuUtilCount := 0
		gpuMemorySum := 0.0
		gpuMemoryCount := 0
		for _, card := range node.CardInfo {
			if card == nil {
				continue
			}
			gpuUtilSum += clampLoad(float64(card.GPU_UTIL) / 100.0)
			gpuUtilCount++
			if usage, ok := gpuMemoryUsage(card.GPU_MEMORY_USED, card.GPU_MEMORY_FREE); ok {
				gpuMemorySum += usage
				gpuMemoryCount++
			}
		}
		if gpuUtilCount > 0 {
			loadParts = append(loadParts, gpuUtilSum/float64(gpuUtilCount))
		}
		if gpuMemoryCount > 0 {
			loadParts = append(loadParts, gpuMemorySum/float64(gpuMemoryCount))
		}
	}

	if len(loadParts) == 0 {
		return 0
	}

	sum := 0.0
	for _, part := range loadParts {
		sum += part
	}
	return sum / float64(len(loadParts))
}

func (monitor *Monitor) clusterAverageNodeLoad(dc int, cl int) float64 {
	cluster := monitor.DataCenterInfo[dc].ClusterInfo[cl]
	if cluster == nil {
		return 0
	}

	sum := 0.0
	count := 0
	for n, node := range cluster.NodeInfo {
		if node == nil || len(node.CardInfo) == 0 {
			continue
		}
		sum += monitor.nodeLoad(dc, cl, n)
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func (monitor *Monitor) applyLoadBalanceFactor(totaltime int64, dc int, cl int, n int) int64 {
	if totaltime <= 0 {
		return totaltime
	}

	avgLoad := monitor.clusterAverageNodeLoad(dc, cl)
	if avgLoad <= 0 {
		return totaltime
	}

	nodeLoad := monitor.nodeLoad(dc, cl, n)
	loadPenalty := LoadBalanceBeta * math.Max(nodeLoad/avgLoad-1, 0)
	adjusted := int64(math.Ceil(float64(totaltime) * (1 + loadPenalty)))
	if adjusted < 1 {
		return 1
	}
	return adjusted
}

func (monitor *Monitor) OptimalAllocate(newJob *Job) bool {
	var optAlc = [5]int{math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt} // optimal Allocation, 存储job的分配位置
	minTotalTime := int64(math.MaxInt64)

	//
	for dc, dataCenterInfo := range monitor.DataCenterInfo {
		for cl, clusterInfo := range dataCenterInfo.ClusterInfo {
			for n, nodeInfo := range clusterInfo.NodeInfo {
				if nodeInfo.CPU_USAGE > 0.7 {
					continue
				}
				if nodeInfo.FREE_MEMORY-10 < newJob.MemoryReq {
					continue
				}
				for c, cardInfo := range nodeInfo.CardInfo {
					transferTime := (newJob.DataSize * 1024) / nodeInfo.Bandwidth
					if cardInfo.GPU_MEMORY_FREE-1024 < newJob.GPUMemoryReq {
						continue
					}
					rawTotaltime := monitor.TotaltimePredict(newJob, dc, cl, n, c)
					if rawTotaltime <= 0 { // 返回了异常值，跳过
						log.Printf("ERROR: RuntimePredict failed at %v %v %v %v, for job %v", dc, cl, n, c, newJob.Batchv1Job.Name)
						continue
					}
					totaltime := monitor.applyLoadBalanceFactor(rawTotaltime, dc, cl, n)
					log.Printf("DEBUG: Totaltime raw=%d adjusted=%d job=%s dc=%d cl=%d n=%d c=%d", rawTotaltime, totaltime, newJob.ID, dc, cl, n, c)
					switch {
					// 如果该节点上存在reservation，且newJob的预计时间超过了reservation time
					case rawTotaltime > monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n].CardInfo[c].ReservedTime && monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n].CardInfo[c].ReservedTime != 0:
						continue
					case totaltime < minTotalTime:
						minTotalTime = totaltime
						optAlc[0] = dc
						optAlc[1] = cl
						optAlc[2] = n
						optAlc[3] = c
						optAlc[4] = int(transferTime)
					}
				}
			}
		}
	}

	// 如果没有合适的位置分配任务
	if optAlc[0] == math.MaxInt && newJob.GPUMemoryReq < 10240 {
		return false
	} else if optAlc[0] == math.MaxInt && newJob.GPUMemoryReq >= 10240 { // 给资源需求量大的作业进行资源预留，避免大作业长时间等待
		return monitor.ReserveAllocate(newJob)
	}

	// 在Job里填写挂载信息
	newJob.DataCenterIDX, newJob.ClusterIDX, newJob.NodeIDX, newJob.CardIDX = optAlc[0], optAlc[1], optAlc[2], optAlc[3]
	log.Println("DEBUG: final alc ", optAlc[0], optAlc[1], optAlc[2], optAlc[3])

	// 分析Job的传输时间
	newJob.TransferTime = int64(optAlc[4])

	// 以下操作应该在发送作业成功的时候进行
	// // 给对应的card上的jobqueue挂上作业
	// monitor.DataCenterInfo[optAlc[0]].ClusterInfo[optAlc[1]].NodeInfo[optAlc[2]].CardInfo[optAlc[3]].JobQueue = append(monitor.DataCenterInfo[optAlc[0]].ClusterInfo[optAlc[1]].NodeInfo[optAlc[2]].CardInfo[optAlc[3]].JobQueue, newJob)

	// // 在对应的Card上，减去该模型预测需要占用的资源 TODO: 目前只考虑了显存
	// monitor.DataCenterInfo[optAlc[0]].ClusterInfo[optAlc[1]].NodeInfo[optAlc[2]].CardInfo[optAlc[3]].GPU_MEMORY_USED += newJob.GPUMemoryReq
	// monitor.DataCenterInfo[optAlc[0]].ClusterInfo[optAlc[1]].NodeInfo[optAlc[2]].CardInfo[optAlc[3]].GPU_MEMORY_FREE -= newJob.GPUMemoryReq
	return true
}

func (monitor *Monitor) ReserveAllocate(newJob *Job) bool {
	var optAlc = [5]int{math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt} // optimal Allocation, 存储job的分配位置
	minTotalTime := int64(math.MaxInt64)
	// 同样模拟计算作业在各个集群的各个节点的各张卡上的运行时间+传输时间，选取时间最少的；但是考虑的情况是该卡为空的；用显存限制来剔除掉不能运行GPU密集型作业的卡；
	// 计算该卡上当前作业的最长运行时间，并记录；如果后续作业想提交到这张卡上，并且运行时间短于该段时间，则准入，否则不准入。
	for dc, dataCenterInfo := range monitor.DataCenterInfo {
		for cl, clusterInfo := range dataCenterInfo.ClusterInfo {
			for n, nodeInfo := range clusterInfo.NodeInfo {
				for c, cardInfo := range nodeInfo.CardInfo {
					transferTime := (newJob.DataSize * 1024) / nodeInfo.Bandwidth
					if cardInfo.GPU_MEMORY_USED+cardInfo.GPU_MEMORY_FREE-1024 < newJob.GPUMemoryReq {
						continue
					}
					if monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n].CardInfo[c].ReservedJob != nil {
						continue
					}
					totaltime := int64(monitor.RandomForestPredict([]string{newJob.JobModelName}, dc, cl, n, c)[0] * float64(newJob.Epoch)) // 计算卡上状态为空时的运行时间
					log.Println("DEBUG: TotaltimeWithoutLoads: ", totaltime, newJob.ID, dc, cl, n, c)
					if totaltime <= 0 { // 返回了异常值，跳过
						log.Printf("ERROR: RuntimePredict failed at %v %v %v %v, for job %v", dc, cl, n, c, newJob.Batchv1Job.Name)
						continue
					}
					if totaltime < minTotalTime {
						minTotalTime = totaltime
						optAlc[0] = dc
						optAlc[1] = cl
						optAlc[2] = n
						optAlc[3] = c
						optAlc[4] = int(transferTime)
					}
				}
			}
		}
	}
	// 如果没有合适的位置分配任务
	if optAlc[0] == math.MaxInt {
		return false
	}

	// 计算该卡上当前作业的最长剩余运行时间 TODO:FIXME:有可能为0，导致出问题
	minRemainedTime := int64(0)
	// minIdx := math.MaxInt64
	for _, job := range monitor.DataCenterInfo[optAlc[0]].ClusterInfo[optAlc[1]].NodeInfo[optAlc[2]].CardInfo[optAlc[3]].JobQueue {
		passed_time := time.Since(job.AssignedTime).Seconds()
		remainedTime := job.TransferTime + int64(float64(job.Epoch)*job.BaselineSpeed) - int64(passed_time)
		if remainedTime > minRemainedTime {
			minRemainedTime = remainedTime
			// minIdx = idx
		}
	}
	monitor.DataCenterInfo[optAlc[0]].ClusterInfo[optAlc[1]].NodeInfo[optAlc[2]].CardInfo[optAlc[3]].ReservedTime = minRemainedTime

	// 在Job里填写挂载信息
	newJob.DataCenterIDX, newJob.ClusterIDX, newJob.NodeIDX, newJob.CardIDX = optAlc[0], optAlc[1], optAlc[2], optAlc[3]
	log.Println("DEBUG: reserved alc ", optAlc[0], optAlc[1], optAlc[2], optAlc[3])

	// 分析Job的传输时间
	newJob.TransferTime = int64(optAlc[4])

	newJob.IsReserved = true
	newJob.ReservedTime = minRemainedTime
	newJob.ReservationStartTime = time.Now()

	// 给对应的card上的ReservedJob挂上
	monitor.DataCenterInfo[optAlc[0]].ClusterInfo[optAlc[1]].NodeInfo[optAlc[2]].CardInfo[optAlc[3]].ReservedJob = newJob
	return true
}
