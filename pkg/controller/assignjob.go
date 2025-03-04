package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
)

func AssignJobToNode(clientset *kubernetes.Clientset, job *Job, nodeName string, namespace string) bool {
	// // 解析YAML为Job对象
	// decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlContent), 4096)
	// var job batchv1.Job
	// if err := decoder.Decode(&job); err != nil {
	// 	return false
	// }

	// 将job中的所有容器，env：NVIDIA_VISIBLE_DEVICES 设置为对应的卡
	for i := range job.Batchv1Job.Spec.Template.Spec.Containers {
		container := &job.Batchv1Job.Spec.Template.Spec.Containers[i]
		flag := false
		for j := range container.Env {
			if container.Env[j].Name == "NVIDIA_VISIBLE_DEVICES" {
				flag = true
				container.Env[j].Value = fmt.Sprint(job.CardIDX)
			}
		}
		if !flag {
			container.Env = append(container.Env, corev1.EnvVar{Name: "NVIDIA_VISIBLE_DEVICES", Value: fmt.Sprint(job.CardIDX)})
		}
	}

	// 设置节点选择器
	if job.Batchv1Job.Spec.Template.Spec.NodeSelector == nil {
		job.Batchv1Job.Spec.Template.Spec.NodeSelector = make(map[string]string)
	}
	job.Batchv1Job.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"] = nodeName

	// // 确定命名空间（默认为default）
	// namespace := job.Namespace
	// if namespace == "" {
	// 	namespace = "default"
	// }

	// 设置annatation 国网环境下hami schduler需要
	if job.Batchv1Job.Spec.Template.Annotations == nil {
		job.Batchv1Job.Spec.Template.Annotations = make(map[string]string)
	}
	job.Batchv1Job.Spec.Template.Annotations["hami.io/resource-pool"] = "poc" // TODO:FIXME:

	// 创建Job
	_, err := clientset.BatchV1().Jobs(namespace).Create(
		context.TODO(),
		&job.Batchv1Job,
		metav1.CreateOptions{},
	)
	if err != nil {
		log.Println("ERROR: AssignJobToNode failed!", err)
		return false
	}
	return true
}

func (monitor *Monitor) AssignJob(job *Job) bool {
	// return true // 测试用
	if monitor.AssignJobWithinController(job) {
		log.Printf("DEBUG: Job %v assigned", job.ID)
		return true
	}
	return false
}

func (monitor *Monitor) AssignJobWithinController(job *Job) bool { // 使用controller内部的方案提交作业
	return AssignJobToNode(monitor.DataCenterInfo[job.DataCenterIDX].ClusterInfo[job.ClusterIDX].ClusterClientSet, job, monitor.DataCenterInfo[job.DataCenterIDX].ClusterInfo[job.ClusterIDX].NodeInfo[job.NodeIDX].NodeID, NAMESPACE)
}

func AssignJobWithSystem(job *Job) bool { // 通过调度器后台来分发作业，由浪潮完成
	return false
}

func (monitor *Monitor) ScheduleAndAssign() {
	var SchduleFailedJob = JobQueue{}
	var AssignedFailedJob = JobQueue{}
	for _, job := range monitor.JobPool.OriginJob {
		monitor.JobAnalyze(job)
		if monitor.OptimalAllocate(job) {
			if job.IsReserved {
				monitor.JobPool.ReservedJob = append(monitor.JobPool.ReservedJob, job)
			} else {
				monitor.JobPool.ScheduledJob = append(monitor.JobPool.ScheduledJob, job)
				if monitor.AssignJob(job) {
					job.AssignedTime = time.Now()
					monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob, job)

					cardInfoPointer := monitor.GetCardInfoPointerFromJob(job)
					// 给对应的card上的jobqueue挂上作业
					cardInfoPointer.JobQueue = append(cardInfoPointer.JobQueue, job)
					// 在对应的Card上，减去该模型预测需要占用的资源 TODO: 目前只考虑了显存
					cardInfoPointer.GPU_MEMORY_USED += job.GPUMemoryReq
					cardInfoPointer.GPU_MEMORY_FREE -= job.GPUMemoryReq
				} else {
					AssignedFailedJob = append(AssignedFailedJob, job)
				}
			}
		} else {
			SchduleFailedJob = append(SchduleFailedJob, job)
		}
	}
	monitor.JobPool.OriginJob = SchduleFailedJob
	log.Println("INFO: SchduleFailedJob", SchduleFailedJob.GetID())
	monitor.JobPool.ScheduledJob = AssignedFailedJob
	log.Println("INFO: AssignedFailedJob", AssignedFailedJob.GetID())
	log.Println("INFO: ReservedJob", monitor.JobPool.ReservedJob.GetID())
	log.Println("INFO: AssignedJob: ", monitor.JobPool.AssignedJob.GetID())
	monitor.JobPool.AssignedJob.List()
}

// 对SchduleFailedJob、AssignedFailedJob以及ReservedJob进行持续处理
func (monitor *Monitor) PersistentPredictor() {
	for {
		// 对AssignedFailedJob(即ScheduledJob）进行重试，若还是失败，重新放回originJob
		for _, job := range monitor.JobPool.ScheduledJob {
			times := 0
			for !monitor.AssignJob(job) && times < 3 {
				times += 1
			}
			switch {
			case times >= 3:
				monitor.JobPool.OriginJob = append(monitor.JobPool.OriginJob, job)
			case times < 3:
				job.AssignedTime = time.Now()
				monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob, job)
			}
		}

		// 对AssignedJob进行监控
		flag := false // 指示是否有AssignedJob已经完成
		finishedJobIdx := []int{}
		for idx, ele := range monitor.JobPool.AssignedJob {
			joblist, _ := jobList(monitor.DataCenterInfo[ele.DataCenterIDX].ClusterInfo[ele.ClusterIDX].ClusterClientSet, NAMESPACE)
			for _, job := range joblist.Items {
				if job.Name == ele.Batchv1Job.Name {
					switch {
					case job.Status.Succeeded == 1:
						flag = true
						log.Printf("INFO:AssignedJob finished, %v %v %v %v %v", ele.ID, ele.DataCenterIDX, ele.ClusterIDX, ele.NodeIDX, ele.CardIDX)
					case job.Status.Failed == 1:
						flag = true
						log.Printf("ERROR:AssignedJob failed, %v %v %v %v %v", ele.ID, ele.DataCenterIDX, ele.ClusterIDX, ele.NodeIDX, ele.CardIDX)
					case job.Status.Active == 1:
						continue
					default:
					}
					finishedJobIdx = append(finishedJobIdx, idx)
					break
				}
			}
		}
		// 删除已经完成的Job
		for i := len(finishedJobIdx) - 1; i >= 0; i-- {
			job := monitor.JobPool.AssignedJob[finishedJobIdx[i]]
			monitor.GetCardInfoPointerFromJob(job).JobQueue.RemoveJob(job.ID)
			monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob[:finishedJobIdx[i]], monitor.JobPool.AssignedJob[finishedJobIdx[i]+1:]...)
		}

		// 当有任务结束时
		if flag {
			// 对ReservedJob进行再分配
			var AssignFailedJobQueue = JobQueue{}
			for _, job := range monitor.JobPool.ReservedJob {
				if int64(time.Since(job.ReservationStartTime).Seconds()) > job.ReservedTime {
					if monitor.AssignJob(job) {
						monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob, job)
						monitor.DataCenterInfo[job.DataCenterIDX].ClusterInfo[job.ClusterIDX].NodeInfo[job.NodeIDX].CardInfo[job.CardIDX].JobQueue = append(monitor.DataCenterInfo[job.DataCenterIDX].ClusterInfo[job.ClusterIDX].NodeInfo[job.NodeIDX].CardInfo[job.CardIDX].JobQueue, job)
						monitor.DataCenterInfo[job.DataCenterIDX].ClusterInfo[job.ClusterIDX].NodeInfo[job.NodeIDX].CardInfo[job.CardIDX].ReservedTime = 0
						monitor.DataCenterInfo[job.DataCenterIDX].ClusterInfo[job.ClusterIDX].NodeInfo[job.NodeIDX].CardInfo[job.CardIDX].ReservedJob = nil
					} else {
						AssignFailedJobQueue = append(AssignFailedJobQueue, job)
					}
				} else {
					AssignFailedJobQueue = append(AssignFailedJobQueue, job)
				}
			}
			monitor.JobPool.ReservedJob = AssignFailedJobQueue

			// 对OriginJob进行分配
			monitor.ScheduleAndAssign()
		}

		time.Sleep(time.Minute)
	}
}
