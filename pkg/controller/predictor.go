package controller

import (
	"cpn-controller/pkg/python"
	"cpn-controller/pkg/utils"
	"log"
	"math"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 5. 在每个集群的每台服务器上运行基准测试程序，获得评价指标（暂定resnet50、yolov8m、llama3，每个各10mins）
// 6. 实现预测器的功能（能够根据提供的模型信息，给出指标），预测其在A100上的平均运行时间
// 7. 模拟分析newJob在某个卡上的运行时间

// 检测每个节点是否已经跑过benchmark
func (monitor *Monitor) checkBenchMark() {
	for _, datacenter := range monitor.DataCenterInfo {
		for _, cluster := range datacenter.ClusterInfo {
			for _, node := range cluster.NodeInfo {
				for _, card := range node.CardInfo {
					if card.BenchMark.Model1AVGRunTime == 0.0 {
						if !monitor.runBenchMark(datacenter.DataCenterID, cluster.ClusterID, node.NodeID, card.CardID) {
							continue
						} else {
							log.Printf("ERROR: No BenchMark in DataCenter: %v ClusterID: %v NodeID: %v CardID: %v", datacenter.DataCenterID, cluster.ClusterID, node.NodeID, card.CardID)
						}
					}
				}
			}
		}
	}
}

// 运行基准测试程序，获得评价指标 TODO: 先在json文件里手动配置，后续增加功能
func (monitor *Monitor) runBenchMark(DataCenterID string, ClusterID string, NodeID string, CardID string) bool {
	return true
}

// 读取并解析model_baseline.csv文件
func (monitor *Monitor) readModelBaseline() {
	// 获取项目工作目录，并读取model_baseline.csv文件
	root, err := utils.GetProjectRoot()
	if err != nil {
		log.Println("ERROR: JobAnalyze faild", err)
	}
	fp := filepath.Join(root, "pkg", "controller", "model_baseline.csv")

	// 解析
	_, lines := utils.ReadCsv(fp)
	var modelBaseline = map[string][]string{}
	for _, ele := range lines {
		modelBaseline[ele[0]] = ele[1:]
	}
	monitor.ModelBaseline = modelBaseline

	fp2 := filepath.Join(root, "pkg", "controller", "model_baseline2.csv")
	_, modelBaseline2 := utils.ReadCsv(fp2)
	// 对三个模型部分的内容进行排序
	for idx, ele := range modelBaseline2[135:] {
		type Pair struct {
			str1 string
			str2 string
		}
		pair1 := Pair{ele[0], ele[1]}
		pair2 := Pair{ele[2], ele[3]}
		pair3 := Pair{ele[4], ele[5]}
		pairs := []Pair{pair1, pair2, pair3}
		sort.Slice(pairs, func(i int, j int) bool { return pairs[i].str1 < pairs[j].str1 })
		modelBaseline2[135+idx] = []string{pairs[0].str1, pairs[0].str2, pairs[1].str1, pairs[1].str2, pairs[2].str1, pairs[2].str2}
	}

	monitor.ModelBaseline2 = modelBaseline2
}

// 作业分析器 分析作业的memoryReq、JobType等数据 TODO:现在都是静态配置，之后可以设计动态配置
func (monitor *Monitor) JobAnalyze(job *Job) {
	if _, exists := monitor.ModelBaseline[job.JobModelName]; exists {
		job.ID = job.JobSpec.Name
		job.GPUMemoryReq, _ = strconv.ParseInt(monitor.ModelBaseline[job.JobModelName][0], 10, 64)
		job.BaselineSpeed, _ = strconv.ParseFloat(monitor.ModelBaseline[job.JobModelName][1], 10)
	} else {
	}

	if job.JobModelName == "llama3" || job.JobModelName == "glm4" || job.JobModelName == "qwen2.5" {
		job.JobType = "GPU"
	} else {
		job.JobType = "CPU"
	}
}

// 预测器逻辑实现，返回预估的作业完成的总时间，即传输+运行时间
func (monitor *Monitor) TotaltimePredict(newJob *Job, dc int, cl int, n int, c int) (runtime int64) {
	startTime := time.Now()
	jobs := [][]int64{{newJob.DataSize * 1024 / monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n].Bandwidth, newJob.Epoch}} // FIXME:第一列为传输时间，第二列为剩余运行epoch
	jobID := []string{newJob.ID}
	jobModelNames := []string{newJob.JobModelName}
	// 分析当前该卡上有的作业，以及其剩余轮次
	for _, job := range monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n].CardInfo[c].JobQueue {
		// 先检测已有job的状态，比如job是否在传输过程中，并计算job的剩余轮次
		var transferRemainTime = int64(math.MaxInt64)                             // 剩余传输时间
		var remainedEpoch = int64(math.MaxInt64)                                  //  剩余运行轮次
		if job.TransferTime > int64(time.Now().Sub(job.AssignedTime).Seconds()) { // 还在传输中
			transferRemainTime = job.TransferTime - int64(time.Now().Sub(job.AssignedTime).Seconds())
			remainedEpoch = job.Epoch
		} else { // 传输已完成 TODO:这部分测试未覆盖到，因为当前算法未作业队列一次性读入并分配位置，不存在传输完成了的情况
			transferRemainTime = int64(0)
			remainedEpoch = int64((float64(job.Epoch)*job.BaselineSpeed - time.Now().Sub(job.AssignedTime).Seconds()) / float64(job.Epoch))
			if remainedEpoch < 0 { // 作业已经完成，跳过
				continue
			}
		}
		jobs = append(jobs, []int64{transferRemainTime, remainedEpoch})
		jobID = append(jobID, job.ID)
		jobModelNames = append(jobModelNames, job.JobModelName)
	}
	// 分析当前作业和已有作业并行时候的runtime
	// // 先按照jobModelNames的顺序进行排序
	// type Pair struct {
	// 	JobModelNames string
	// 	Jobs          []int64
	// 	JobID         string
	// }
	// pairs := make([]Pair, len(jobModelNames))
	// for i := 0; i < len(jobModelNames); i++ {
	// 	pairs[i] = Pair{
	// 		JobModelNames: jobModelNames[i],
	// 		Jobs:          jobs[i],
	// 		JobID:         jobID[i],
	// 	}
	// }
	// // 对 Pair 切片按照 Str 字段进行排序
	// sort.Slice(pairs, func(i, j int) bool {
	// 	return pairs[i].JobModelNames < pairs[j].JobModelNames
	// })
	// // 从排序后的 Pair 切片中提取出 jobs 元素，组成新的 int 切片
	// sortedJobs := make([][]int64, len(pairs))
	// sortJobID := make([]string, len(pairs))
	// for i := 0; i < len(pairs); i++ {
	// 	sortedJobs[i] = pairs[i].Jobs
	// 	sortJobID[i] = pairs[i].JobID
	// }
	// jobs = sortedJobs
	// jobID = sortJobID

	// newBaseline := monitor.RealDataPredict(jobModelNames)
	newBaseline := monitor.RandomForestPredict(jobModelNames, dc, cl, n, c)

	// // 分析该作业的预计运行时间 未考虑部分作业完成后的运行速度
	// for idx, jmn := range jobModelNames {
	// 	if jmn == newJob.JobModelName {
	// 		return int64(newBaseline[idx] * float64(newJob.Epoch))
	// 	}
	// }

	// 先找到最先结束的Job，循环分析该Job结束后剩余的Job的运行时间，直到Job全部运行完
	totalTime := int64(0)
	for len(jobs) > 0 {
		minRemainedTime := math.MaxFloat64
		minRemainedIDX := math.MaxInt
		for idx, nbl := range newBaseline {
			if float64(jobs[idx][0])+nbl*float64(jobs[idx][1]) < minRemainedTime { //表达式为 传输时间+运行时间
				minRemainedTime = float64(jobs[idx][0]) + nbl*float64(jobs[idx][1])
				minRemainedIDX = idx
			}
		}
		totalTime += int64(minRemainedTime)
		if jobID[minRemainedIDX] == newJob.ID {
			return totalTime
		} else {
			// 移除已经结束的作业
			jobs = append(jobs[:minRemainedIDX], jobs[minRemainedIDX+1:]...)
			jobID = append(jobID[:minRemainedIDX], jobID[minRemainedIDX+1:]...)
			jobModelNames = append(jobModelNames[:minRemainedIDX], jobModelNames[minRemainedIDX+1:]...)
			newBaseline = append(newBaseline[:minRemainedIDX], newBaseline[minRemainedIDX+1:]...)

			// 更新当前作业的剩余epoch
			for i := range jobs {
				if jobs[i][0]-totalTime > 0 { //还在传输过程中
					jobs[i][0] -= totalTime
				} else { //传输完成  TODO:这部分的测试也未覆盖
					jobs[i][0] = 0
					partRuntime := totalTime - jobs[i][0] // 作业已经执行的时间
					jobs[i][1] -= int64(float64(partRuntime) * newBaseline[i])
					if jobs[i][1] < 0 {
						jobs[i][1] = 0
					}
				}
			}
		}
		// 重新分析多作业并行的情况
		newBaseline = monitor.RealDataPredict(jobModelNames)
		newBaseline = monitor.RandomForestPredict(jobModelNames, dc, cl, n, c)
	}
	log.Println("job predict time consumed:", time.Now().Sub(startTime).Seconds())
	return 0 // 以秒为单位
}

func NewRandomForestPredictor() bool {
	root, _ := utils.GetProjectRoot()
	fp := filepath.Join(root, `pkg`, `python`, `socket_server.py`)
	cmd := exec.Command("python", fp)

	err := cmd.Start()
	if err != nil {
		log.Println("ERROR: NewRandomForestPredictor faild", err)
		return false
	}

	return true
}

func (monitor *Monitor) RandomForestPredict(jobModelNames []string, dc int, cl int, n int, c int) []float64 {
	benchMark := monitor.DataCenterInfo[dc].ClusterInfo[cl].NodeInfo[n].CardInfo[c].BenchMark
	bm := []string{
		strconv.FormatFloat(benchMark.Model1AVGRunTime, 'f', -1, 64),
		strconv.FormatFloat(benchMark.Model2AVGRunTime, 'f', -1, 64),
		strconv.FormatFloat(benchMark.Model3AVGRunTime, 'f', -1, 64),
	}
	str_response := python.SendStruct(`rfp.sock`, bm, jobModelNames)
	response := []float64{}
	strs := strings.Split(str_response, ",")
	// for _, str := range strs {
	// 	value, _ := strconv.ParseFloat(str, 64)
	// 	if value < 0.01 {
	// 		response = append(response, 0.0)
	// 	} else {
	// 		response = append(response, value)
	// 	}
	// }
	for i := range jobModelNames {
		value, _ := strconv.ParseFloat(strs[i], 64)
		response = append(response, value)
	}
	log.Printf("DEBUG: jobModelNames:%v response:%v", jobModelNames, response)
	return response
}

// 预测器算法实现（从实际数据中获取运行时间）, 返回当前所有模型的单epoch运行时间
func (monitor *Monitor) RealDataPredict(jobModelNames []string) []float64 {
	if len(jobModelNames) == 1 {
		for _, ele := range monitor.ModelBaseline2 {
			if ele[0] == jobModelNames[0] {
				num, _ := strconv.ParseFloat(ele[1], 10)
				return []float64{num}
			}
		}
	} else if len(jobModelNames) == 2 {
		for _, ele := range monitor.ModelBaseline2 {
			if ele[0] == jobModelNames[0] && ele[2] == jobModelNames[1] {
				num1, _ := strconv.ParseFloat(ele[1], 10)
				num2, _ := strconv.ParseFloat(ele[3], 10)
				return []float64{num1, num2}
			}
			if ele[0] == jobModelNames[1] && ele[2] == jobModelNames[0] {
				num2, _ := strconv.ParseFloat(ele[1], 10)
				num1, _ := strconv.ParseFloat(ele[3], 10)
				return []float64{num1, num2}
			}
		}
	} else if len(jobModelNames) == 3 {
		// 对jobModelNames进行排序，方便比对
		sort.Strings(jobModelNames)
		for _, ele := range monitor.ModelBaseline2[135:] {
			if ele[0] == jobModelNames[0] && ele[2] == jobModelNames[1] && ele[4] == jobModelNames[2] {
				num1, _ := strconv.ParseFloat(ele[1], 10)
				num2, _ := strconv.ParseFloat(ele[3], 10)
				num3, _ := strconv.ParseFloat(ele[5], 10)
				return []float64{num1, num2, num3}
			}
		}
	}
	return nil
}

func (monitor *Monitor) InitPredictor() { // TODO:FIXME:random forest 需要优化，llama3那块还要检测
	monitor.readModelBaseline()
	if !NewRandomForestPredictor() {
		log.Println("ERROR: NewRandomForestPredictor failed")
	}
	var SchduleFailedJob = []*Job{}
	var AssignedFailedJob = []*Job{}
	for _, job := range monitor.JobPool.OriginJob {
		monitor.JobAnalyze(job)
		if monitor.OptimalAllocate(job) {
			monitor.JobPool.ScheduledJob = append(monitor.JobPool.ScheduledJob, job)
			if AssignJobWithSystem(job) {
				job.AssignedTime = time.Now()
				monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob, job)
			} else {
				AssignedFailedJob = append(AssignedFailedJob, job)
			}
		} else {
			SchduleFailedJob = append(SchduleFailedJob, job)
		}
	}
	monitor.JobPool.OriginJob = SchduleFailedJob
	if len(SchduleFailedJob) > 0 {
		log.Println("ERROR: SchduleFailedJob", SchduleFailedJob)
	}
	monitor.JobPool.ScheduledJob = AssignedFailedJob
	if len(AssignedFailedJob) > 0 {
		log.Println("ERROR: AssignedFailedJob", AssignedFailedJob)
	}
	log.Println("INFO: AssignedJob: ", monitor.JobPool.AssignedJob.GetID())
	monitor.JobPool.AssignedJob.List()

}
