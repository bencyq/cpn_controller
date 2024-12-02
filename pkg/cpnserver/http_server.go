package cpnserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// 消息队列 负责收集http服务器接受到的信息
var MsgQueue = make(chan []byte, 100)

// 接收http请求
func handler(w http.ResponseWriter, r *http.Request) {
	// 只接受 POST 请求
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体中的数据
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	MsgQueue <- body
	fmt.Fprintln(w, "Message received")
}

// 创建一个http服务器
func newHttpServer() {
	http.HandleFunc("/", handler)
	go func() {
		err := http.ListenAndServe(ServerIP, nil)
		if err != nil {
			log.Fatal("Error starting server: ", err)
		}
	}()
	log.Printf("Listening on %v...\n", ServerIP)
	log.Println("HttpServer succeed!")
}

// 负责http_server的业务逻辑
func APP1() {
	// 创建http服务端
	go newHttpServer()

	// 接受信息 TODO:并发送给调度器
	for msg := range MsgQueue {
		var mergedMap map[string]interface{}
		_ = json.Unmarshal(msg, &mergedMap)
		log.Printf("get msg from cluster: %v", mergedMap["client-name"])
		HeartBeat <- mergedMap
	}
}
