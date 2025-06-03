package fifo

import (
	"cpn-controller/pkg/controller"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// 对于所有任务，按先进先出的顺序分配；按卡的序号依次分配，只要显存够用，就分配上去
// 逻辑为：逐个作业在卡之间依次分配，遍历完所有卡后若还有未分配的作业，则从头开始再次遍历卡
// 给作业添加RestartPolicy，避免作业因OOM失败
// 结束后手动检查作业是否因为OOM报错
// TODO: 应当设置定时重启OOM作业

var NAMESPACE = `fifo`

func FifoSchedule(monitor *controller.Monitor) {
	// TODO: 每个作业依次遍历所有位置，放置后减去显存
	var ScheduleFailedJob = make(controller.JobQueue, 0)
	for {
		jobIdx := 0
		monitor.GetMetric(3)
		for dc, dataCenterInfo := range monitor.DataCenterInfo {
			for cl, clusterInfo := range dataCenterInfo.ClusterInfo {
				for n, nodeInfo := range clusterInfo.NodeInfo {
					if nodeInfo.CPU_USAGE > 0.9 {
						continue
					}
					if nodeInfo.FREE_MEMORY < 1024 {
						continue
					}
					for c, cardInfo := range nodeInfo.CardInfo {
						if jobIdx >= len(monitor.JobPool.OriginJob) {
							return
						}
						job := monitor.JobPool.OriginJob[jobIdx]
						monitor.JobAnalyze(job)
						if cardInfo.GPU_MEMORY_USED+cardInfo.GPU_MEMORY_FREE-1024 < job.GPUMemoryReq {
							continue
						}
						if (job.Batchv1Job.Annotations[`model_name`] == `llama3` || job.Batchv1Job.Annotations[`model_name`] == `glm4` || job.Batchv1Job.Annotations[`model_name`] == `qwen2.5`) && cardInfo.CardModel == `Tesla P100-PCIE-16GB` { //设置一下LLM作业不上p100
							continue
						}
						job.DataCenterIDX, job.ClusterIDX, job.NodeIDX, job.CardIDX = dc, cl, n, c
						job.Batchv1Job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
						if monitor.AssignJobToNode(clusterInfo.ClusterClientSet, job, nodeInfo.NodeID, controller.NAMESPACE) {
							log.Printf("INFO: Job %v assigned to %v %v %v %v", job.ID, dc, cl, n, c)
							cardInfo.GPU_MEMORY_USED += job.GPUMemoryReq
							cardInfo.GPU_MEMORY_FREE -= job.GPUMemoryReq
							monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob, job)
							jobIdx++
							if jobIdx >= len(monitor.JobPool.OriginJob) {
								monitor.JobPool.OriginJob = controller.JobQueue{}
								return
							}
							continue
						}
					}
				}
			}
		}
		ScheduleFailedJob = append(ScheduleFailedJob, monitor.JobPool.OriginJob[jobIdx:]...)
		if len(ScheduleFailedJob) > 0 {
			monitor.JobPool.OriginJob = ScheduleFailedJob
			ScheduleFailedJob = controller.JobQueue{}
		} else {
			return
		}
		time.Sleep(time.Second * 20)
	}
}

func MonitorAssignedJob(monitor *controller.Monitor) {
	for len(monitor.JobPool.AssignedJob) > 0 || len(monitor.JobPool.OriginJob) > 0 {
		FifoSchedule(monitor)
		// 对AssignedJob进行监控
		finishedJobIdx := []int{}
		// log.Println("INFO: Start monitor AssignedJob")
		for idx, ele := range monitor.JobPool.AssignedJob {
			joblist, _ := controller.JobList(monitor.DataCenterInfo[ele.DataCenterIDX].ClusterInfo[ele.ClusterIDX].ClusterClientSet, controller.NAMESPACE)
			for _, job := range joblist.Items {
				if job.Name == ele.Batchv1Job.Name {
					switch {
					case job.Status.Succeeded == 1:
						log.Printf("INFO: AssignedJob finished, %v %v %v %v %v, runtime %v", ele.ID, ele.DataCenterIDX, ele.ClusterIDX, ele.NodeIDX, ele.CardIDX, time.Since(job.Status.StartTime.Time).Seconds())
					case job.Status.Failed == 1:
						monitor.DeleteJobFromNode(monitor.GetClusterInfoPointerFromJob(ele).ClusterClientSet, ele, NAMESPACE)
						monitor.JobPool.OriginJob = append(monitor.JobPool.OriginJob, ele)
						log.Printf("ERROR: AssignedJob failed, job deleted %v %v %v %v %v", ele.ID, ele.DataCenterIDX, ele.ClusterIDX, ele.NodeIDX, ele.CardIDX)
					case job.Status.Active == 1:
						continue
					}
					finishedJobIdx = append(finishedJobIdx, idx)
					break
				}
			}
		}
		// 删除已经完成的Job
		for i := len(finishedJobIdx) - 1; i >= 0; i-- {
			monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob[:finishedJobIdx[i]], monitor.JobPool.AssignedJob[finishedJobIdx[i]+1:]...)
		}

		// // 重新提交失败的作业
		// var ScheduleFailedJob = controller.JobQueue{}
		// for _, job := range monitor.JobPool.OriginJob {
		// 	if !monitor.AssignJobToNode(monitor.GetClusterInfoPointerFromJob(job).ClusterClientSet, job, monitor.GetNodeInfoPointerFromJob(job).NodeID, NAMESPACE) {
		// 		ScheduleFailedJob = append(ScheduleFailedJob, job)
		// 	} else {
		// 		monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob, job)
		// 		log.Printf("INFO: Reassign job %v %v %v %v %v", job.ID, job.DataCenterIDX, job.ClusterIDX, job.NodeIDX, job.CardIDX)
		// 	}

		// }
		// monitor.JobPool.OriginJob = ScheduleFailedJob
		log.Println("INFO: Sleep")
		time.Sleep(time.Second * 40)
	}
}
