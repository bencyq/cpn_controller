package python

// 负责创建socket，和pyhon进程通信

import (
	"cpn-controller/pkg/utils"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
)

// 采用socket与py进程通信
func SendStruct(socketName string, elements ...interface{}) string {
	// socket 路径
	root, _ := utils.GetProjectRoot()
	fp := filepath.Join(root, "pkg", "python", socketName)
	// conn, err := net.Dial("unix", socketPath)
	conn, err := net.DialTimeout("unix", fp, time.Minute*5) // 设定超时时间为5mins
	if err != nil {
		cwd, _ := os.Getwd()
		log.Printf("ERROR: Failed to connect: %v\tcurrent working directory: %v", err, cwd)
	}
	defer conn.Close()

	// 发送数据给 Python 进程
	// for _, ele := range elements {

	// }
	msg, err := json.Marshal(elements)
	_, err = conn.Write([]byte(msg))
	if err != nil {
		log.Printf("ERROR: Failed to write data: %v", err)
	}
	// fmt.Println("Sent to Python:", msg)

	// 接收 Python 进程的响应，并直接返回数据
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		log.Printf("ERROR: Failed to read data: %v", err)
	}
	return string(buf[:n])
	// fmt.Println("Received from Python:", string(buf[:n]))
}
