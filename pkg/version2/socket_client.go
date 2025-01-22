package version2

// 负责创建socket，和pyhon进程通信

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

// 发送monitor整个的配置信息，接受作业队列，预测器也集成到调度器的py实现里
func (monitor *Monitor) clientForScheduler() {
	// 创建连接
	socketPath := "./scheduler.sock"
	// conn, err := net.Dial("unix", socketPath)
	conn, err := net.DialTimeout("unix", socketPath, time.Minute*5) // 设定超时时间为5mins
	if err != nil {
		cwd, _ := os.Getwd()
		log.Printf("ERROR: Failed to connect: %v\tcurrent working directory: %v", err, cwd)
	}
	defer conn.Close()

	// 发送数据给 Python 进程
	msg, err := json.Marshal(monitor)
	_, err = conn.Write([]byte(msg))
	if err != nil {
		log.Printf("ERROR: Failed to write data: %v", err)
	}
	// fmt.Println("Sent to Python:", msg)

	// 接收 Python 进程的响应，把接收到的信息加入ScheduledJob中TODO:
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		log.Printf("ERROR: Failed to read data: %v", err)
	}
	fmt.Println("Received from Python:", string(buf[:n]))
}
