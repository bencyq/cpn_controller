package controller

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

func TestBuildPredictorArgsIncludesBenchmarkTimes(t *testing.T) {
	card := &CardInfo{
		SM_ACTIVE:    11.1,
		SM_OCCUPANCY: 22.2,
		DRAM_ACTIVE:  33.3,
		BenchMark: BenchMark{
			Model1AVGRunTime: 0.014736412,
			Model2AVGRunTime: 0.051585681,
		},
	}

	args := buildPredictorArgs("densenet121", card)
	want := []string{
		"densenet121",
		"--sm-active", "11.100000",
		"--sm-occupancy", "22.200000",
		"--dram-active", "33.300000",
		"--compute-benchmark-time", "0.014736",
		"--memory-benchmark-time", "0.051586",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected predictor args: got %v want %v", args, want)
	}
}

func TestInitPredictor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	monitor := NewMonitor()
	monitor.InitPredictor(ctx)
	// monitor.PersistentPredictor()
}

func TestRealDataPredict(t *testing.T) {
	monitor := NewMonitor()
	monitor.ReadModelBaseline()
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
