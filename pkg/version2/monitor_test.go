package version2

import (
	"io"
	"log"
	"os"
	"testing"
)

func TestUnmarshalJson(t *testing.T) {
	fileName := "example.json"
	// 打开文件
	file, err := os.Open(fileName)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		log.Println("Error reading file:", err)
		return
	}
	unmarshalJson(content)
}
