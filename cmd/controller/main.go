package main

import (
	"context"
	"cpn-controller/pkg/controller"
	"log"
	"time"
)

func main() {
	start_time := time.Now()

	// 初始化流程
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	controller.JsonUrl = "example3.json"
	monitor := controller.NewMonitor()
	monitor.InitPredictor(ctx)

	monitor.PersistentPredictor()
	log.Println("INFO: Finished!")
	log.Println("INFO: Time consumed:", time.Since(start_time).Minutes())
}
