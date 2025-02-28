package main

import (
	"context"
	"cpn-controller/pkg/controller"
)

func main() {
	// 初始化流程
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	monitor := controller.NewMonitor()
	monitor.InitPredictor(ctx)
	monitor.PersistentPredictor()
}
