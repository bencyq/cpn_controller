package cpnserver

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	batchv1 "k8s.io/api/batch/v1"
)

type Scheduler struct {
	JobPool
	cluster  []Cluster
	Strategy // 排序策略，对JobPool里的作业进行分析，并加入ClusterJobQueue中
}

func (s *Scheduler) FindCluster(clusterName string) (c *Cluster, err error) {
	cl := &s.cluster
	for i := range s.cluster {
		if (*cl)[i].name == clusterName {
			return &(*cl)[i], nil
		}
	}
	// return nil, errors.New(fmt.Sprintf("cannot find cluster %v", clusterName))
	return nil, fmt.Errorf("cannot find cluster %v", clusterName)
}

type Job struct {
	// filepath     string
	Timestamp    time.Time          `json:"timestamp"`      // 作业提交的时间
	PresumedTime string             `json:"presumedTime"`   // 预计完成需要时间
	Pre_job      string             `json:"pre_job"`        // 记录Job队列中前面的Job的名字，帮助判断该Job是否可以被发送给客户端
	Lock         []chan interface{} `json:"lock,omitempty"` // 用来确保Job在合适时候发送给客户端
}

// JobPool 管理作业池，存储作业并提供增删操作
type JobPool map[string]Job

type Card struct {
	id              int
	GPU_UTIL        float64
	GPU_MEMORY_FREE int64
	GPU_MEMORY_USED int64
}

type Node struct {
	name         string
	card         []Card
	CPU_USAGE    float64
	TOTAL_MEMORY int64
	FREE_MEMORY  int64
	CARDMODEL    CardModel
}

type ClusterJobQueue struct {
	JobQueue chan Job
}

type Cluster struct {
	name   string
	ipPort string
	node   []Node
	ClusterJobQueue
	bandwidth int // 带宽，单位为MB/s
}

func (c *Cluster) FindNode(nodeName string) (n *Node, err error) {
	no := &c.node
	for i := range c.node {
		if (*no)[i].name == nodeName {
			return &(*no)[i], nil
		}
	}
	// return nil, errors.New(fmt.Sprintf("cannot find cluster %v", nodeName))
	return nil, fmt.Errorf("cannot find cluster %v", nodeName)
}

type ClusterJobInterface interface {
	[]ClusterJobQueue
	ListClusters()
	GetJobQueue()
}

type Strategy interface {
}

// 从对应的JobQueue里去除
func finished_job(jobName string) {
	// TODO:
}

var HeartBeat = make(chan map[string]interface{}, 10) // 存放cluster里收集来的信息
var joblist *batchv1.JobList

// 分析heartbeat信息
func (sch *Scheduler) HeartBeatAnalyse() (err error) {
	for heartbeat := range HeartBeat {
		cluster_name, _ := heartbeat["client-name"].(string)
		// 反序列joblist，分析有哪些Job已经完成，从JobQueue里去除
		data, _ := json.Marshal(heartbeat["job"])
		_ = json.Unmarshal(data, &joblist)
		for _, ele := range joblist.Items {
			if ele.Status.Succeeded == 1 {
				finished_job(ele.Name) // TODO:
			}
		}
		// 找到指定的cluster
		c, err := sch.FindCluster(cluster_name)
		if err != nil {
			log.Println("Error:", err)
			return err
		}
		promMetrics, ok := heartbeat["prom"].(map[string]interface{})
		if ok {
			for nodeName, _ := range promMetrics {
				nodeMetrics, ok := promMetrics[nodeName].(map[string]interface{})
				if ok {
					// 找到指定的node
					node, err := c.FindNode(nodeName)
					if err != nil {
						log.Println("Error:", err)
					}
					data, ok := nodeMetrics["CPU_USAGE"].([]interface{})
					if ok {
						tmp, ok := data[0].(string)
						if ok {
							node.CPU_USAGE, _ = strconv.ParseFloat(tmp, 64)
						}
					}
					data, ok = nodeMetrics["TOTAL_MEMORY"].([]interface{})
					if ok {
						tmp, ok := data[0].(string)
						if ok {
							node.TOTAL_MEMORY, _ = strconv.ParseInt(tmp, 10, 0)
						}
					}
					data, ok = nodeMetrics["FREE_MEMORY"].([]interface{})
					if ok {
						tmp, ok := data[0].(string)
						if ok {
							node.FREE_MEMORY, _ = strconv.ParseInt(tmp, 10, 0)
						}
					}
					card_metic, ok := nodeMetrics["GPU_UTIL"].([]interface{})
					if ok {
						for idx, content := range card_metic {
							node.card[idx].GPU_UTIL, _ = strconv.ParseFloat(content.(string), 64)
						}
					}
					card_metic, ok = nodeMetrics["GPU_MEMORY_FREE"].([]interface{})
					if ok {
						for idx, content := range card_metic {
							node.card[idx].GPU_MEMORY_FREE, _ = strconv.ParseInt(content.(string), 10, 0)
						}
					}
					card_metic, ok = nodeMetrics["GPU_MEMORY_USED"].([]interface{})
					if ok {
						for idx, content := range card_metic {
							node.card[idx].GPU_MEMORY_USED, _ = strconv.ParseInt(content.(string), 10, 0)
						}
					}
				}
			}
		}
	}
	return
}

func (jp *JobPool) InitJobPool(inputFilePath string, outputFilePath string) (err error) {

	os.MkdirAll(inputFilePath, 0744)  // 作业提交的目录
	os.MkdirAll(outputFilePath, 0744) // 已完成作业的存档目录
	os.MkdirAll("tmpyaml", 0744)      // 存放已经加入JobPool的yaml缓存目录
	file, _ := os.OpenFile("tmpyaml/jobpoolcache.json", os.O_RDWR|os.O_CREATE, 0744)
	defer file.Close()
	// 从缓存目录以及缓存信息中读取上次中断的JobPool信息
	cache, _ := os.ReadFile("tmpyaml/jobpoolcache.json")
	if len(cache) != 0 {
		json.Unmarshal(cache, jp)
	}

	// 每10s扫描一次，将提交的文件加入JobPool中，并缓存文件以及JobPool信息
	go func() {
		for {
			dirEntry, _ := os.ReadDir(inputFilePath)
			for _, ele := range dirEntry {
				if ele.IsDir() {
					continue
				}
				os.Rename(inputFilePath+"/"+ele.Name(), "tmpyaml/"+ele.Name())
				(*jp)[ele.Name()] = Job{
					Timestamp: time.Now(),
				}
			}
			cache, _ := json.Marshal(jp)
			file.Write(cache)

			time.Sleep(10 * time.Second)
		}
	}()

	return nil
}

func Run() (err error) {
	// 初始化Cluster
	sch := Scheduler{
		cluster: []Cluster{cluster_one},
	}

	// 初始化JobPool
	sch.JobPool = make(JobPool)
	sch.JobPool.InitJobPool("yamlexample", "yamlarchive")

	// 分析heartbeat信息
	go sch.HeartBeatAnalyse()

	// TODO: 算法分析当前集群的状态，分配作业队列
	// TODO: 实现跨集群的数据迁移

	return nil
}
