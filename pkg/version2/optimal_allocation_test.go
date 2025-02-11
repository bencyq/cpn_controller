package version2

import "testing"

func TestAnalyze(t *testing.T) {
	monitor := NewMonitor()
	for _, job := range monitor.JobPool.OriginJobQueue {
		monitor.OptimalAllocate(job)
	}
}
