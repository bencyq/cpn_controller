package cpnserver

import (
	"encoding/json"

	batchv1 "k8s.io/api/batch/v1"
)

type Scheduler struct {
	JobPool
	cluster  []Cluster
	Strategy // 排序策略，对JobPool里的作业进行分析，并加入ClusterJobQueue中
}

var HeartBeat = make(chan map[string]interface{}, 10) // 存放cluster里收集来的信息

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
	id int
}

type Node struct {
	name string
	card []Card
}

type ClusterJobQueue struct {
	JobQueue chan Job
}

type Cluster struct {
	name   string
	ipPort string
	node   []Node
	ClusterJobQueue
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

func (*Scheduler) Run() {
	// 初始化Cluster
	sch := Scheduler{
		cluster: []Cluster{cluster_one},
	}

	// 分析heartbeat信息
	for heartbeat := range HeartBeat {
		cluster_name := heartbeat["client-name"]

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
			for metric, content := range promMetrics {

			}
		}
	}
}
