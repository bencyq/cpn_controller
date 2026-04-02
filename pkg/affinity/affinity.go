package affinity

import (
	"cpn-controller/pkg/controller"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	modelZooPath        = "/data/cyq/hase/model_zoo/models"
	maxNodeCPUUsage     = 0.9
	minNodeFreeMemory   = 1024
	gpuMemoryReserveMB  = 1024
	affinityWeight      = 0.7
	memoryPackingWeight = 0.3
)

type ResourceClass string

const (
	ComputeIntensive ResourceClass = "compute-intensive"
	MemoryIntensive  ResourceClass = "memory-intensive"
)

// EvaluationMetrics 预留了三类评价维度，便于后续对不同调度策略做统一对比。
type EvaluationMetrics struct {
	TimeEfficiency      float64
	ResourceUtilization float64
	SystemStability     float64
}

type ModelProfile struct {
	Family        string
	ResourceClass ResourceClass
}

type OfflineProfiler struct {
	Profiles       map[string]ModelProfile
	AffinityMatrix map[ResourceClass]map[ResourceClass]float64
}

type affinityTarget struct {
	dataCenterIdx int
	clusterIdx    int
	nodeIdx       int
	cardIdx       int
	score         float64
}

var (
	profilerOnce sync.Once
	profilerInst *OfflineProfiler
)

func getOfflineProfiler() *OfflineProfiler {
	profilerOnce.Do(func() {
		profilerInst = buildOfflineProfiler(modelZooPath)
	})
	return profilerInst
}

func buildOfflineProfiler(modelPath string) *OfflineProfiler {
	profiles := make(map[string]ModelProfile)
	entries, err := os.ReadDir(modelPath)
	if err != nil {
		log.Printf("ERROR: affinity read model zoo failed: %v", err)
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			family := normalizeModelFamily(entry.Name())
			if family == "" {
				continue
			}
			profiles[family] = ModelProfile{
				Family:        family,
				ResourceClass: classifyFamily(family),
			}
		}
	}

	// 目录不可读或为空时，保底给出当前已知模型族分类，避免调度逻辑直接失效。
	for family, class := range defaultFamilyClasses() {
		if _, exists := profiles[family]; exists {
			continue
		}
		profiles[family] = ModelProfile{
			Family:        family,
			ResourceClass: class,
		}
	}

	return &OfflineProfiler{
		Profiles: profiles,
		AffinityMatrix: map[ResourceClass]map[ResourceClass]float64{
			ComputeIntensive: {
				ComputeIntensive: 0.35,
				MemoryIntensive:  1.0,
			},
			MemoryIntensive: {
				ComputeIntensive: 1.0,
				MemoryIntensive:  0.35,
			},
		},
	}
}

func defaultFamilyClasses() map[string]ResourceClass {
	return map[string]ResourceClass{
		"alexnet":      MemoryIntensive,
		"densenet121":  MemoryIntensive,
		"densenet169":  MemoryIntensive,
		"densenet201":  MemoryIntensive,
		"mobilenet_v2": MemoryIntensive,
		"mobilenet_v3": MemoryIntensive,
		"resnet18":     ComputeIntensive,
		"resnet50":     ComputeIntensive,
		"resnet152":    ComputeIntensive,
		"vgg11":        ComputeIntensive,
		"vgg13":        ComputeIntensive,
		"vgg16":        ComputeIntensive,
		"vgg19":        ComputeIntensive,
	}
}

func normalizeModelFamily(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return ""
	}

	name = strings.TrimSuffix(name, filepath.Ext(name))
	if idx := strings.Index(name, "_bs"); idx >= 0 {
		name = name[:idx]
	}
	return name
}

func classifyFamily(family string) ResourceClass {
	family = normalizeModelFamily(family)
	switch {
	case strings.HasPrefix(family, "resnet"), strings.HasPrefix(family, "vgg"):
		return ComputeIntensive
	case strings.HasPrefix(family, "mobilenet"), strings.HasPrefix(family, "densenet"), strings.HasPrefix(family, "alexnet"):
		return MemoryIntensive
	case strings.HasPrefix(family, "llama"), strings.HasPrefix(family, "qwen"), strings.HasPrefix(family, "glm"):
		return MemoryIntensive
	default:
		return ComputeIntensive
	}
}

func (profiler *OfflineProfiler) getResourceClass(job *controller.Job) ResourceClass {
	family := normalizeModelFamily(job.JobModelName)
	if profile, exists := profiler.Profiles[family]; exists {
		return profile.ResourceClass
	}
	if job.GPUMemoryReq >= 8192 {
		return MemoryIntensive
	}
	return classifyFamily(family)
}

func isGPUAffinityJob(job *controller.Job, profiler *OfflineProfiler) bool {
	if job.GPUMemoryReq > 0 || job.JobType == "GPU" {
		return true
	}
	family := normalizeModelFamily(job.JobModelName)
	_, exists := profiler.Profiles[family]
	return exists
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

func cardFeasible(card *controller.CardInfo, job *controller.Job) bool {
	return card.GPU_MEMORY_FREE >= job.GPUMemoryReq+gpuMemoryReserveMB
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

func cardCapacity(card *controller.CardInfo) float64 {
	total := card.GPU_MEMORY_FREE + card.GPU_MEMORY_USED
	if total <= 0 {
		return 1
	}
	return float64(total)
}

func memoryPackingScore(card *controller.CardInfo, job *controller.Job) float64 {
	remaining := card.GPU_MEMORY_FREE - job.GPUMemoryReq
	if remaining < 0 {
		remaining = 0
	}
	return 1 - clamp01(float64(remaining)/cardCapacity(card))
}

func (profiler *OfflineProfiler) pairAffinityScore(left *controller.Job, right *controller.Job) float64 {
	leftFamily := normalizeModelFamily(left.JobModelName)
	rightFamily := normalizeModelFamily(right.JobModelName)
	if leftFamily != "" && leftFamily == rightFamily {
		return 0.2
	}

	leftClass := profiler.getResourceClass(left)
	rightClass := profiler.getResourceClass(right)
	if scoreMap, exists := profiler.AffinityMatrix[leftClass]; exists {
		if score, exists := scoreMap[rightClass]; exists {
			return score
		}
	}
	return 0.5
}

func (profiler *OfflineProfiler) cardAffinityScore(card *controller.CardInfo, job *controller.Job) float64 {
	if len(card.JobQueue) == 0 {
		return 0.6
	}

	total := 0.0
	validJobs := 0
	for _, runningJob := range card.JobQueue {
		if runningJob == nil {
			continue
		}
		total += profiler.pairAffinityScore(job, runningJob)
		validJobs++
	}
	if validJobs == 0 {
		return 0.6
	}
	return total / float64(validJobs)
}

func (profiler *OfflineProfiler) rankCard(monitor *controller.Monitor, job *controller.Job, dc int, cl int, n int, c int) float64 {
	card := monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n].CardInfo[c]
	affinityScore := profiler.cardAffinityScore(card, job)
	packingScore := memoryPackingScore(card, job)
	return affinityWeight*affinityScore + memoryPackingWeight*packingScore
}

func AffinityAllocate(monitor *controller.Monitor, job *controller.Job) bool {
	profiler := getOfflineProfiler()
	best := affinityTarget{
		dataCenterIdx: -1,
		clusterIdx:    -1,
		nodeIdx:       -1,
		cardIdx:       -1,
		score:         -math.MaxFloat64,
	}

	for dc, dataCenterInfo := range monitor.DataCenterInfo {
		for cl, clusterInfo := range dataCenterInfo.ClusterInfo {
			for n, nodeInfo := range clusterInfo.NodeInfo {
				if !nodeFeasible(nodeInfo) {
					continue
				}
				for c, cardInfo := range nodeInfo.CardInfo {
					if !cardFeasible(cardInfo, job) {
						continue
					}
					score := profiler.rankCard(monitor, job, dc, cl, n, c)
					if score > best.score {
						best = affinityTarget{
							dataCenterIdx: dc,
							clusterIdx:    cl,
							nodeIdx:       n,
							cardIdx:       c,
							score:         score,
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

func AffinityReady(monitor *controller.Monitor, job *controller.Job) bool {
	return isGPUAffinityJob(job, getOfflineProfiler())
}

func ModelProfiles() []ModelProfile {
	profiler := getOfflineProfiler()
	profiles := make([]ModelProfile, 0, len(profiler.Profiles))
	for _, profile := range profiler.Profiles {
		profiles = append(profiles, profile)
	}
	sort.Slice(profiles, func(i int, j int) bool {
		return profiles[i].Family < profiles[j].Family
	})
	return profiles
}

func affinityAllocateReady(monitor *controller.Monitor, job *controller.Job) bool {
	if !AffinityReady(monitor, job) {
		return false
	}
	return AffinityAllocate(monitor, job)
}

func Run(monitor *controller.Monitor, opts controller.StrategyOptions) {
	if opts.Name == "" {
		opts.Name = "affinity"
	}
	if opts.Namespace == "" {
		opts.Namespace = "affinity"
	}
	monitor.RunStrategy(opts, affinityAllocateReady)
}

func AffinitySchedule(monitor *controller.Monitor) {
	Run(monitor, controller.StrategyOptions{
		Name:      "affinity",
		Namespace: "affinity",
	})
}

func MonitorAssignedJob(monitor *controller.Monitor) {
	AffinitySchedule(monitor)
}
