package controller

import (
	"fmt"
	"testing"
)

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
		fmt.Println(monitor.RandomForestPredict([]string{`llama3`, `densenet121`}, 0, 0, 1, 0))
		fmt.Println(monitor.RandomForestPredict([]string{`llama3`}, 0, 0, 1, 0))
	}

}

func TestRandomForestPredict2(t *testing.T) {
	monitor := NewMonitor()
	fmt.Println(monitor.RandomForestPredict([]string{`llama3`, `densenet121`}, 0, 0, 1, 0))
	fmt.Println(monitor.RandomForestPredict([]string{`llama3`}, 0, 0, 1, 0))
}

func TestRuntimePredict(t *testing.T) {
	monitor := NewMonitor()
	monitor.InitPredictor()
}
