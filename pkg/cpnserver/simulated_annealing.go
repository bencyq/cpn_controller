package cpnserver

import (
	"math/rand"
	"time"
)

// 模拟退火算法
func simulatedAnnealing() {

}

// 计算当前状态的能量（总运行时间）
func calculateEnergy() {

}

// 生成一个邻域状态
func generateNeighbor() {

}

// TODO:获取llama3、sam等在A100上的显卡利用率
func initSA(sch Scheduler) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	for _, job := range sch.JobPool {
		// 随机选择一个集群和节点
		job.ClusterID = r.Intn(len(sch.cluster))
		job.NodeID = r.Intn(len(sch.cluster[job.ClusterID].node))

		if job.JobType == `GPU` {
			job.CardID = r.Intn(len(sch.cluster[job.ClusterID].node[job.NodeID].card))
		} else {
			job.CardID = -1
		}
	}
}
