package controller

import (
	"math"
	"testing"
)

func TestAnalyze(t *testing.T) {
	monitor := NewMonitor()
	for _, job := range monitor.JobPool.OriginJob {
		monitor.OptimalAllocate(job)
	}
}

func TestApplyLoadBalanceFactor(t *testing.T) {
	oldBeta := LoadBalanceBeta
	LoadBalanceBeta = 1.0
	defer func() {
		LoadBalanceBeta = oldBeta
	}()

	monitor := &Monitor{
		DataCenterInfo: []*DataCenterInfo{
			{
				ClusterInfo: []*ClusterInfo{
					{
						NodeInfo: []*NodeInfo{
							{
								CPU_USAGE:    0.2,
								TOTAL_MEMORY: 100,
								FREE_MEMORY:  80,
								CardInfo: []*CardInfo{
									{GPU_UTIL: 20, GPU_MEMORY_USED: 20, GPU_MEMORY_FREE: 80},
								},
							},
							{
								CPU_USAGE:    0.8,
								TOTAL_MEMORY: 100,
								FREE_MEMORY:  20,
								CardInfo: []*CardInfo{
									{GPU_UTIL: 80, GPU_MEMORY_USED: 80, GPU_MEMORY_FREE: 20},
								},
							},
						},
					},
				},
			},
		},
	}

	if load := monitor.nodeLoad(0, 0, 0); math.Abs(load-0.2) > 1e-9 {
		t.Fatalf("unexpected cold node load: %v", load)
	}
	if load := monitor.nodeLoad(0, 0, 1); math.Abs(load-0.8) > 1e-9 {
		t.Fatalf("unexpected hot node load: %v", load)
	}

	coldScore := monitor.applyLoadBalanceFactor(100, 0, 0, 0)
	hotScore := monitor.applyLoadBalanceFactor(100, 0, 0, 1)

	if coldScore != 100 {
		t.Fatalf("expected cold node score to remain 100, got %d", coldScore)
	}
	if hotScore <= 100 {
		t.Fatalf("expected hot node score to increase, got %d", hotScore)
	}
	if hotScore <= coldScore {
		t.Fatalf("expected hot node score %d to be greater than cold node score %d", hotScore, coldScore)
	}
}
