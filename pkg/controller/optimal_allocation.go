package controller

import (
	"log"
	"math"
	"time"
)

//  作业队列按照FIFO顺序排列
// 对于一个作业，模拟其在各个集群运行的完成时间，并进行分配

// 对于一个作业，模拟其在各个集群运行的完成时间
func (monitor *Monitor) OptimalAllocate(newJob *Job) bool {
	var optAlc = [5]int{math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt} // optimal Allocation, 存储job的分配位置
	minTotalTime := int64(math.MaxInt64)

	// 模拟计算作业在各个集群的各个节点的各张卡上的运行时间+传输时间，选取时间最少的
	for dc, dataCenterInfo := range monitor.DataCenterInfo {
		for cl, clusterInfo := range dataCenterInfo.ClusterInfo {
			for n, nodeInfo := range clusterInfo.NodeInfo {
				if nodeInfo.CPU_USAGE > 0.7 {
					continue
				}
				if nodeInfo.FREE_MEMORY-10 < newJob.MemoryReq {
					continue
				}
			overloop:
				for c, cardInfo := range nodeInfo.CardInfo {
					transferTime := (newJob.DataSize * 1024) / nodeInfo.Bandwidth
					if cardInfo.GPU_MEMORY_FREE-1024 < newJob.GPUMemoryReq || len(cardInfo.JobQueue) >= 3 {
						continue
					}
					for _, job := range cardInfo.JobQueue {
						if job.JobType == `GPU` && newJob.JobType == `GPU` {
							continue overloop
						}
					}
					totaltime := monitor.TotaltimePredict(newJob, dc, cl, n, c)
					log.Println("DEBUG: Totaltime: ", totaltime, newJob.ID, dc, cl, n, c)
					if totaltime <= 0 { // 返回了异常值，跳过
						log.Printf("ERROR: RuntimePredict failed at %v %v %v %v, for job %v", dc, cl, n, c, newJob.Batchv1Job.Name)
						continue
					}
					switch {
					// 如果该节点上存在reservation，且newJob的预计时间超过了reservation time
					case totaltime > monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n].CardInfo[c].ReservedTime && monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n].CardInfo[c].ReservedTime != 0:
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
	if optAlc[0] == math.MaxInt && newJob.JobType != "GPU" {
		return false
	} else if optAlc[0] == math.MaxInt && newJob.JobType == "GPU" { // 给资源需求量大的作业进行资源预留，避免大作业长时间等待
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
					if monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n].CardInfo[c].ReservedTime != 0 {
						continue
					}
					for _, job := range cardInfo.JobQueue {
						if job.JobType == `GPU` && newJob.JobType == `GPU` {
							continue
						}
					}
					totaltime := int64(monitor.RandomForestPredict([]string{newJob.JobModelName}, dc, cl, n, c)[0]) * newJob.Epoch // 计算卡上状态为空时的运行时间
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

	// 计算该卡上当前作业的最长剩余运行时间
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
