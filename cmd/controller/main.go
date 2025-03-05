package main

import (
	"context"
	"cpn-controller/pkg/controller"
)

func main() {
	// 初始化流程
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	controller.JsonUrl="example3.json"
	monitor := controller.NewMonitor()
	monitor.InitPredictor(ctx)

	// 每隔一分钟更新一次metric TODO:正式版上线
	// go func() {
	// 	for {
	// 		time.Sleep(time.Minute)
	// 		monitor.getMetric()
	// 	}
	// }()

	monitor.PersistentPredictor()
}
