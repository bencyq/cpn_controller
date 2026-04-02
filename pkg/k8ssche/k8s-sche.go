package k8ssche

import (
	"cpn-controller/pkg/controller"
	"log"
	"math"
	"sort"
)

const (
	maxNodeCPUUsage     = 0.9
	minNodeFreeMemory   = 1024
	nodeMemoryReserveMB = 1024
	gpuMemoryReserveMB  = 1024
	balanceWeight       = 0.5
	gpuResourceWeight   = 0.3
	affinityWeight      = 0.2
)

type nodeCandidate struct {
	dataCenterIdx int
	clusterIdx    int
	nodeIdx       int
	totalScore    float64
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

func positiveOrOne(v int64) float64 {
	if v <= 0 {
		return 1
	}
	return float64(v)
}

func nodeGPUStats(node *controller.NodeInfo) (totalFree int64, totalCapacity int64) {
	for _, card := range node.CardInfo {
		totalFree += card.GPU_MEMORY_FREE
		totalCapacity += card.GPU_MEMORY_FREE + card.GPU_MEMORY_USED
	}
	return totalFree, totalCapacity
}

func nodePassesFilter(node *controller.NodeInfo, job *controller.Job) bool {
	if node.CPU_USAGE > maxNodeCPUUsage {
		return false
	}

	requiredNodeMemory := int64(minNodeFreeMemory)
	if job.MemoryReq > 0 && job.MemoryReq+nodeMemoryReserveMB > requiredNodeMemory {
		requiredNodeMemory = job.MemoryReq + nodeMemoryReserveMB
	}
	if node.FREE_MEMORY < requiredNodeMemory {
		return false
	}

	if job.JobType != "GPU" {
		return true
	}

	totalGPUFree, _ := nodeGPUStats(node)
	return totalGPUFree >= job.GPUMemoryReq+gpuMemoryReserveMB
}

// modelAffinityScore 用一个很轻量的近似来模拟 kube-scheduler 的亲和性打分。
// 若节点上已有相同模型作业，则加分；空闲节点给中性分；其余节点给较低分。
func modelAffinityScore(node *controller.NodeInfo, job *controller.Job) float64 {
	hasRunningJob := false
	for _, card := range node.CardInfo {
		for _, runningJob := range card.JobQueue {
			hasRunningJob = true
			if runningJob.JobModelName == job.JobModelName {
				return 1.0
			}
		}
	}
	if !hasRunningJob {
		return 0.6
	}
	return 0.3
}

func scoreNode(node *controller.NodeInfo, job *controller.Job) float64 {
	totalGPUFree, totalGPUCapacity := nodeGPUStats(node)

	effectiveNodeMemory := node.FREE_MEMORY - minNodeFreeMemory
	if job.MemoryReq > 0 {
		effectiveNodeMemory = node.FREE_MEMORY - job.MemoryReq
	}
	if effectiveNodeMemory < 0 {
		effectiveNodeMemory = 0
	}
	nodeMemoryScore := clamp01(float64(effectiveNodeMemory) / positiveOrOne(node.TOTAL_MEMORY))

	remainingGPUFree := totalGPUFree
	if job.JobType == "GPU" {
		remainingGPUFree -= job.GPUMemoryReq
	}
	if remainingGPUFree < 0 {
		remainingGPUFree = 0
	}
	gpuMemoryScore := clamp01(float64(remainingGPUFree) / positiveOrOne(totalGPUCapacity))

	balanceScore := 1 - math.Abs(nodeMemoryScore-gpuMemoryScore)
	affinityScore := modelAffinityScore(node, job)

	return balanceWeight*balanceScore +
		gpuResourceWeight*gpuMemoryScore +
		affinityWeight*affinityScore
}

func rankCandidateNodes(monitor *controller.Monitor, job *controller.Job) []nodeCandidate {
	candidates := make([]nodeCandidate, 0)
	for dc, dataCenterInfo := range monitor.DataCenterInfo {
		for cl, clusterInfo := range dataCenterInfo.ClusterInfo {
			for n, nodeInfo := range clusterInfo.NodeInfo {
				if !nodePassesFilter(nodeInfo, job) {
					continue
				}
				candidates = append(candidates, nodeCandidate{
					dataCenterIdx: dc,
					clusterIdx:    cl,
					nodeIdx:       n,
					totalScore:    scoreNode(nodeInfo, job),
				})
			}
		}
	}

	sort.SliceStable(candidates, func(i int, j int) bool {
		if candidates[i].totalScore == candidates[j].totalScore {
			if candidates[i].dataCenterIdx != candidates[j].dataCenterIdx {
				return candidates[i].dataCenterIdx < candidates[j].dataCenterIdx
			}
			if candidates[i].clusterIdx != candidates[j].clusterIdx {
				return candidates[i].clusterIdx < candidates[j].clusterIdx
			}
			return candidates[i].nodeIdx < candidates[j].nodeIdx
		}
		return candidates[i].totalScore > candidates[j].totalScore
	})
	return candidates
}

// selectCardForNode 只是为了兼容当前 controller 的提交流程。
// 节点的过滤和打分都保持在 node 级别，不用单卡指标参与排序。
func selectCardForNode(node *controller.NodeInfo, job *controller.Job) (int, bool) {
	if job.JobType != "GPU" {
		if len(node.CardInfo) == 0 {
			return -1, false
		}
		return 0, true
	}

	for idx, card := range node.CardInfo {
		if card.GPU_MEMORY_FREE >= job.GPUMemoryReq+gpuMemoryReserveMB {
			return idx, true
		}
	}
	return -1, false
}

// K8sScheduleAllocate 参考 kube-scheduler 的 Filter/Score 流程，
// 先在节点级别筛选候选节点并打分，再把结果映射到当前 controller 需要的卡级索引。
func K8sScheduleAllocate(monitor *controller.Monitor, job *controller.Job) bool {
	candidates := rankCandidateNodes(monitor, job)
	for _, candidate := range candidates {
		node := monitor.DataCenterInfo[candidate.dataCenterIdx].
			ClusterInfo[candidate.clusterIdx].
			NodeInfo[candidate.nodeIdx]

		cardIdx, ok := selectCardForNode(node, job)
		if !ok {
			log.Printf(
				"INFO: k8s-sche node %v %v %v passed node-level score but no single GPU can fit job %v",
				candidate.dataCenterIdx,
				candidate.clusterIdx,
				candidate.nodeIdx,
				job.ID,
			)
			continue
		}

		job.DataCenterIDX = candidate.dataCenterIdx
		job.ClusterIDX = candidate.clusterIdx
		job.NodeIDX = candidate.nodeIdx
		job.CardIDX = cardIdx
		return true
	}
	return false
}

func Run(monitor *controller.Monitor, opts controller.StrategyOptions) {
	if opts.Name == "" {
		opts.Name = "k8s-sche"
	}
	if opts.Namespace == "" {
		opts.Namespace = "k8s-sche"
	}
	monitor.RunStrategy(opts, K8sScheduleAllocate)
}

func K8sSchedule(monitor *controller.Monitor) {
	Run(monitor, controller.StrategyOptions{
		Name:      "k8s-sche",
		Namespace: "k8s-sche",
	})
}

func MonitorAssignedJob(monitor *controller.Monitor) {
	K8sSchedule(monitor)
}
