package controller

import "testing"

func TestInitPredictor(t *testing.T) {
	NewMonitor().InitPredictor()
}

func TestRealDataPredict(t *testing.T) {
	monitor := NewMonitor()
	monitor.readModelBaseline()
	monitor.RealDataPredict([]string{`densenet169`})
	monitor.RealDataPredict([]string{`densenet169`, `resnet18`})
	monitor.RealDataPredict([]string{`resnet152`, `yolov8x`, `densenet121`})
}

func TestRandomForestPredict(t *testing.T) {
	if NewRandomForestPredictor() {
		monitor := NewMonitor()
		monitor.RandomForestPredict([]string{`test`})
	}

}

func TestRuntimePredict(t *testing.T) {
	monitor := NewMonitor()
	monitor.InitPredictor()
}
