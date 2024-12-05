package cpnserver

import (
	"encoding/json"
	"strconv"

	batchv1 "k8s.io/api/batch/v1"
)

type Scheduler struct {
	JobPool
	cluster    []Cluster
	Strategy                        // 排序策略，对JobPool里的作业进行分析，并加入ClusterJobQueue中
	ClusterMap *map[string]*Cluster // 方便根据clustername快速查找cluster
}

func (s *Scheduler) InitializeClusterMap() {
	// s.ClusterMap = &map[string]Cluster{}
	// // 遍历 Scheduler 中的 cluster 列表，将每个 Cluster 的 name 索引到 map 中
	// for _, cluster := range s.cluster {
	// 	(*s.ClusterMap)[cluster.name] = cluster
	// }
	s.ClusterMap = &map[string]*Cluster{} // 使用指针存储 Cluster
	// 遍历 Scheduler 中的 cluster 列表，将每个 Cluster 的 name 索引到 map 中
	for i := range s.cluster {
		// 将 Cluster 的指针存储到 ClusterMap 中
		(*s.ClusterMap)[s.cluster[i].name] = &s.cluster[i]
	}
}

func (s *Scheduler) InitializeNodeMap() {
	for i := range s.cluster {
		cl := &s.cluster[i]
		cl.NodeMap = &map[string]*Node{}
		// 遍历 Scheduler 中的 cluster 列表，将每个 Cluster 的 name 索引到 map 中
		for _, node := range cl.node {
			(*cl.NodeMap)[node.name] = &node
		}

	}
}

type Job struct {
	filepath     string
	presumedTime string             // 预计完成需要时间
	pre_job      string             // 记录Job队列中前面的Job的名字，帮助判断该Job是否可以被发送给客户端
	Lock         []chan interface{} // 用来确保Job在合适时候发送给客户端
}

// JobPool 管理作业池，存储作业并提供增删操作
type JobPool struct {
	jobs map[string]Job
}

type JobPoolInterface interface {
	Add(job Job)
	Delete(jobName string)
	List()
	Init() // 负责初始化JobPool里的内容
}

func (jp *JobPool) Add(job Job, jobName string) {
	jp.jobs[jobName] = job
}

func (jp *JobPool) Delete(jobName string) {
	delete(jp.jobs, jobName)
}

func (jp *JobPool) List() {

}

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
}

type ClusterJobQueue struct {
	JobQueue chan Job
}

type Cluster struct {
	name   string
	ipPort string
	node   []Node
	ClusterJobQueue
	NodeMap *map[string]*Node
}

// func (cl *Cluster) InitializeNodeMap() {
// 	cl.NodeMap = &map[string]Node{}
// 	// 遍历 Scheduler 中的 cluster 列表，将每个 Cluster 的 name 索引到 map 中
// 	for _, node := range cl.node {
// 		(*cl.NodeMap)[node.name] = node
// 	}
// }

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

func Run() {
	// 初始化Cluster
	sch := Scheduler{
		cluster: []Cluster{cluster_one},
	}
	sch.InitializeClusterMap()
	// for i := range sch.cluster {
	// 	cl := &sch.cluster[i]
	// 	cl.InitializeNodeMap()
	// }
	sch.InitializeNodeMap()
	// 分析heartbeat信息
	for heartbeat := range HeartBeat {
		cluster_name, _ := heartbeat["client-name"].(string)
		// 反序列joblist，分析有哪些Job已经完成，从JobQueue里去除
		var joblist *batchv1.JobList
		data, _ := json.Marshal(heartbeat["job"])
		_ = json.Unmarshal(data, &joblist)
		for _, ele := range joblist.Items {
			if ele.Status.Succeeded == 1 {
				finished_job(ele.Name)
			}
		}
		promMetrics, ok := heartbeat["prom"].(map[string]interface{})
		if ok {
			for nodeName, _ := range promMetrics {
				nodeMetrics, ok := promMetrics[nodeName].(map[string]interface{})
				if ok {
					node, ok := (*(*sch.ClusterMap)[cluster_name].NodeMap)[nodeName]
					if ok {
						// node.CPU_USAGE = nodeMetrics["CPU_USAGE"].(string)
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
						for idx, content := range card_metic {
							node.card[idx].GPU_UTIL, _ = strconv.ParseFloat(content.(string), 64)
						}
						card_metic, ok = nodeMetrics["GPU_MEMORY_FREE"].([]interface{})
						for idx, content := range card_metic {
							node.card[idx].GPU_MEMORY_FREE, _ = strconv.ParseInt(content.(string), 10, 0)
						}
						card_metic, ok = nodeMetrics["GPU_MEMORY_USED"].([]interface{})
						for idx, content := range card_metic {
							node.card[idx].GPU_MEMORY_USED, _ = strconv.ParseInt(content.(string), 10, 0)
						}
					}
				}
			}
		}

	}
}
