package cpnserver

import (
	"encoding/json"
	"log"
	"testing"
)

func TestRun(t *testing.T) {
	// 创建http服务端
	go newHttpServer()

	// 接受信息,并发送给调度器
	go func() {
		for msg := range MsgQueue {
			var mergedMap map[string]interface{}
			_ = json.Unmarshal(msg, &mergedMap)
			log.Printf("get msg from cluster: %v", mergedMap["client-name"])
			HeartBeat <- mergedMap
		}
	}()
	Run()
}
