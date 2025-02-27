package utils

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/yaml"
)

func GetProjectRoot() (string, error) {
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

// type JobQueue []batchv1.Job

func MakeRandomJobQueue(directoryIn, directoryOut string) {
	JobQueue := []batchv1.Job{}
	entries, _ := os.ReadDir(directoryIn)
	for _, entry := range entries {
		file, _ := os.Open(directoryIn + `/` + entry.Name())
		defer file.Close()
		content, _ := io.ReadAll(file)
		job := batchv1.Job{}
		yaml.Unmarshal(content, &job)
		JobQueue = append(JobQueue, job)
	}

	// 将切片随机扩展
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))
	NewJobQueue := []batchv1.Job{}
	for _, job := range JobQueue {
		times := randSrc.Intn(3) + 1
		for i := 0; i < times; i += 1 {
			NewJobQueue = append(NewJobQueue, job)
		}
	}
	JobQueue = NewJobQueue

	// 打乱顺序
	rand.Shuffle(len(JobQueue), func(i, j int) {
		JobQueue[i], JobQueue[j] = JobQueue[j], JobQueue[i]
	})

	// 按照顺序修改Job的ID，并随机Job的epoch和Datasize
	for idx, job := range JobQueue {
		job.Name = fmt.Sprint(idx)
		job.Annotations[`data_size`] = fmt.Sprint(randSrc.Intn(20) + 10)
		if job.Annotations[`model_name`] == `llama3` || job.Annotations[`model_name`] == `qwen2.5` || job.Annotations[`model_name`] == `glm4` {
			job.Annotations[`epoch`] = fmt.Sprint(randSrc.Intn(100) + 100)
		} else {
			job.Annotations[`epoch`] = fmt.Sprint(randSrc.Intn(5000) + 10000)
		}

		// 将Job写入目标地址，文件名为序号
		content, _ := yaml.Marshal(job)
		os.WriteFile(directoryOut+`/`+job.Name+`.yaml`, content, 0644)
	}

}
