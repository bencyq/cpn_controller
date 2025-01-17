package version2

import (
	"io"
	"log"
	"os"
	"testing"
)

func getJsonWithFile(fileName string) (content []byte) {
	// 打开文件
	file, err := os.Open(fileName)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// 读取文件内容
	content, err = io.ReadAll(file)
	if err != nil {
		log.Println("Error reading file:", err)
		return
	}
	return content
}

func TestUnmarshalJson(t *testing.T) {
	var root Root
	root.unmarshalJson(getJsonWithFile("example.json"))
}

func TestGetMetric(t *testing.T) {
	var root Root
	// 初始化root
	root.unmarshalJson(getJsonWithFile("example2.json"))
	root.getMetric()
}
