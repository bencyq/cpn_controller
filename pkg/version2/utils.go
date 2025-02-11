package version2

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func getProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			return currentDir, nil
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("go.mod not found")
		}
		currentDir = parentDir
	}
}

func ReadCsv(fp string) ([]string, [][]string) {
	file, err := os.Open(fp)
	if err != nil {
		log.Println("ERROR: ReadCsv:", err)
		return nil, nil
	}
	defer file.Close()

	// 创建一个CSV读取器
	reader := csv.NewReader(file)

	// 读取第一行作为表头
	headers, err := reader.Read()
	if err != nil {
		log.Println("ERROR: ReadCsv:", err)
		return nil, nil
	}
	// 读取其余行作为数据
	lines, err := reader.ReadAll()
	if err != nil {
		log.Println("ERROR: ReadCsv:", err)
		return nil, nil
	}
	return headers, lines
}
