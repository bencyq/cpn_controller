package mbf

import (
	"cpn-controller/pkg/controller"
	"math"
)

const (
	maxNodeCPUUsage    = 0.9
	minNodeFreeMemory  = 1024
	gpuMemoryReserveMB = 1024
)

type bestFitTarget struct {
	dataCenterIdx int
	clusterIdx    int
	nodeIdx       int
	cardIdx       int
	remainingMB   int64
}

func nodeFeasible(node *controller.NodeInfo) bool {
	if node.CPU_USAGE > maxNodeCPUUsage {
		return false
	}
	if node.FREE_MEMORY < minNodeFreeMemory {
		return false
	}
	return true
}

func cardRemainingMemory(card *controller.CardInfo, job *controller.Job) (int64, bool) {
	effectiveFree := card.GPU_MEMORY_FREE - gpuMemoryReserveMB
	if effectiveFree < job.GPUMemoryReq {
		return 0, false
	}
	return effectiveFree - job.GPUMemoryReq, true
}

// BestFitAllocate 只负责做 MBF 选卡，不会直接向 K8s 提交作业。
func BestFitAllocate(monitor *controller.Monitor, job *controller.Job) bool {
	best := bestFitTarget{
		dataCenterIdx: -1,
		clusterIdx:    -1,
		nodeIdx:       -1,
		cardIdx:       -1,
		remainingMB:   math.MaxInt64,
	}

	for dc, dataCenterInfo := range monitor.DataCenterInfo {
		for cl, clusterInfo := range dataCenterInfo.ClusterInfo {
			for n, nodeInfo := range clusterInfo.NodeInfo {
				if !nodeFeasible(nodeInfo) {
					continue
				}
				for c, cardInfo := range nodeInfo.CardInfo {
					remainingMB, ok := cardRemainingMemory(cardInfo, job)
					if !ok {
						continue
					}
					if remainingMB < best.remainingMB {
						best = bestFitTarget{
							dataCenterIdx: dc,
							clusterIdx:    cl,
							nodeIdx:       n,
							cardIdx:       c,
							remainingMB:   remainingMB,
						}
					}
				}
			}
		}
	}

	if best.dataCenterIdx < 0 {
		return false
	}

	job.DataCenterIDX = best.dataCenterIdx
	job.ClusterIDX = best.clusterIdx
	job.NodeIDX = best.nodeIdx
	job.CardIDX = best.cardIdx
	return true
}

func Run(monitor *controller.Monitor, opts controller.StrategyOptions) {
	if opts.Name == "" {
		opts.Name = "mbf"
	}
	if opts.Namespace == "" {
		opts.Namespace = "mbf"
	}
	monitor.RunStrategy(opts, BestFitAllocate)
}

func MBFSchedule(monitor *controller.Monitor) {
	Run(monitor, controller.StrategyOptions{
		Name:      "mbf",
		Namespace: "mbf",
	})
}

func MonitorAssignedJob(monitor *controller.Monitor) {
	MBFSchedule(monitor)
}
