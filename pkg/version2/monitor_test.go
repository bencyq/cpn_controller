package version2

import (
	"testing"
)

func TestUnmarshalJson(t *testing.T) {
	var monitor Monitor
	monitor.unmarshalJson(getJsonWithFile("example.json"))
}

func TestGetMetric(t *testing.T) {
	var monitor Monitor
	// 初始化monitor
	monitor.unmarshalJson(getJsonWithFile("example2.json"))
	monitor.getMetric()
}

func TestGetJob(t *testing.T) {
	var monitor Monitor
	monitor.getJob()
}

func TestNewMonitor(t *testing.T) {
	NewMonitor()
}

func TestMainLogic(t *testing.T) { // monitor主逻辑
	monitor := NewMonitor()
	monitor.InitPredictor()
}
