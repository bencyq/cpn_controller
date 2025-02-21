package controller

import (
	"context"
	"fmt"
	"testing"
)

func TestInitPredictor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	NewMonitor().InitPredictor(ctx)
}

func TestRealDataPredict(t *testing.T) {
	monitor := NewMonitor()
	monitor.readModelBaseline()
	monitor.RealDataPredict([]string{`densenet169`})
	monitor.RealDataPredict([]string{`densenet169`, `resnet18`})
	monitor.RealDataPredict([]string{`resnet152`, `yolov8x`, `densenet121`})
}

func TestNewRandomForestPredictor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	NewRandomForestPredictor(ctx)

}

func TestRandomForestPredict(t *testing.T) {
	ctx := context.Background()
	if NewRandomForestPredictor(ctx) {
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
	ctx := context.Background()
	monitor := NewMonitor()
	monitor.InitPredictor(ctx)
}
