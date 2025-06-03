package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

func (monitor *Monitor) AssignJobToNode(clientset *kubernetes.Clientset, job *Job, nodeName string, namespace string) bool {
	// // 解析YAML为Job对象
	// decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlContent), 4096)
	// var job batchv1.Job
	// if err := decoder.Decode(&job); err != nil {
	// 	return false
	// }

	for i := range job.Batchv1Job.Spec.Template.Spec.Containers {
		container := &job.Batchv1Job.Spec.Template.Spec.Containers[i]
		flag1 := false
		flag2 := false
		for j := range container.Env {
			// 国网Hami环境下使用mps，必须能见所有卡，不能用NVIDIA_VISIBLE_DEVICES，改用CUDA_VISIBLE_DEVICES

			// 不申请nvidia.com/gpu，不走hami scheduler，适应国网的当前环境，使用NVIDIA_VISIBLE_DEVICES=all来可见所有卡，用CUDA_VISIBLE_DEVICES来指定卡
			if container.Env[j].Name == "NVIDIA_VISIBLE_DEVICES" {
				flag1 = true
				container.Env[j].Value = "all"
			}
			if container.Env[j].Name == "CUDA_VISIBLE_DEVICES" {
				flag2 = true
				container.Env[j].Value = fmt.Sprint(job.CardIDX)
			}
		}
		if !flag1 {
			// container.Env = append(container.Env, corev1.EnvVar{Name: "NVIDIA_VISIBLE_DEVICES", Value: fmt.Sprint(job.CardIDX)})
			container.Env = append(container.Env, corev1.EnvVar{Name: "NVIDIA_VISIBLE_DEVICES", Value: `all`})

		}
		if !flag2 {
			// container.Env = append(container.Env, corev1.EnvVar{Name: "NVIDIA_VISIBLE_DEVICES", Value: fmt.Sprint(job.CardIDX)})
			container.Env = append(container.Env, corev1.EnvVar{Name: "CUDA_VISIBLE_DEVICES", Value: fmt.Sprint(job.CardIDX)})

		}

		// 不申请nvidia.com/gpu，不走hami scheduler，适应国网的当前环境

		// // 设置resources:limits:nvidia.com/gpu: 为物理卡的数量（hami环境需要这样才能正常运行mps）
		// if container.Resources.Limits == nil {
		// 	container.Resources.Limits = make(corev1.ResourceList)
		// }
		// nodeInfo := monitor.GetNodeInfoPointerFromJob(job)
		// container.Resources.Limits[`nvidia.com/gpu`] = *resource.NewQuantity(int64(nodeInfo.CardNums), resource.DecimalSI)
		// container.Resources.Limits[`nvidia.com/gpumem`] = *resource.NewQuantity(int64(nodeInfo.CardNums)*(nodeInfo.CardInfo[0].GPU_MEMORY_USED+nodeInfo.CardInfo[0].GPU_MEMORY_FREE), resource.DecimalSI)

		// 加入limits: k8s.amazonaws.com/vgpu: 1
		if container.Resources.Limits == nil {
			container.Resources.Limits = make(corev1.ResourceList)
		}
		container.Resources.Limits[`k8s.amazonaws.com/vgpu`] = *resource.NewQuantity(int64(1), resource.DecimalExponent)

		// 将Job中的epoch写入yaml中
		flagEpoch := false
		for _, ele := range container.Args {
			if ele == "--epoch" {
				flagEpoch = true
			}
		}
		if !flagEpoch {
			container.Args = append(container.Args, "--epoch", fmt.Sprint(job.Epoch))
		}

		// 添加挂载
		flag3 := false
		if container.VolumeMounts == nil {
			container.VolumeMounts = []corev1.VolumeMount{}
		} else {
			for idx := range container.VolumeMounts {
				if container.VolumeMounts[idx].Name == `nvidia-mps` {
					container.VolumeMounts[idx].MountPath = `/tmp/nvidia-mps`
					flag3 = true
				}
			}
		}
		if !flag3 {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{Name: `nvidia-mps`, MountPath: `/tmp/nvidia-mps`})
		}
	}

	// 设置节点选择器
	if job.Batchv1Job.Spec.Template.Spec.NodeSelector == nil {
		job.Batchv1Job.Spec.Template.Spec.NodeSelector = make(map[string]string)
	}
	job.Batchv1Job.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"] = nodeName

	// // 设置annatation 国网环境下hami schduler需要
	// if job.Batchv1Job.Spec.Template.Annotations == nil {
	// 	job.Batchv1Job.Spec.Template.Annotations = make(map[string]string)
	// }
	// job.Batchv1Job.Spec.Template.Annotations["hami.io/resource-pool"] = "poc"

	// 设置IPChost为true
	if !job.Batchv1Job.Spec.Template.Spec.HostIPC {
		job.Batchv1Job.Spec.Template.Spec.HostIPC = true
	}

	// 设置挂载
	flag4 := false
	if job.Batchv1Job.Spec.Template.Spec.Volumes == nil {
		job.Batchv1Job.Spec.Template.Spec.Volumes = []corev1.Volume{}
	} else {
		for idx := range job.Batchv1Job.Spec.Template.Spec.Volumes {
			if job.Batchv1Job.Spec.Template.Spec.Volumes[idx].Name == `nvidia-mps` {
				job.Batchv1Job.Spec.Template.Spec.Volumes[idx].VolumeSource = corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: `/tmp/nvidia-mps`}}
				flag4 = true
			}
		}
	}
	if !flag4 {
		job.Batchv1Job.Spec.Template.Spec.Volumes = append(job.Batchv1Job.Spec.Template.Spec.Volumes, corev1.Volume{Name: `nvidia-mps`, VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: `/tmp/nvidia-mps`}}})
	}

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

	// 先验证卡上有足够的显存
	if job.GPUMemoryReq > monitor.GetCardInfoPointerFromJob(job).GPU_MEMORY_FREE {
		log.Printf("DEBUG: Job %v GPUMemoryReq not satisfied, %v %v %v %v", job.ID, job.DataCenterIDX, job.ClusterIDX, job.NodeIDX, job.CardIDX)
		return false
	}
	if monitor.AssignJobWithinController(job) {
		log.Printf("INFO: Job %v assigned, %v %v %v %v", job.ID, job.DataCenterIDX, job.ClusterIDX, job.NodeIDX, job.CardIDX)
		return true
	}
	return false
}

func (monitor *Monitor) AssignJobWithinController(job *Job) bool { // 使用controller内部的方案提交作业
	return monitor.AssignJobToNode(monitor.DataCenterInfo[job.DataCenterIDX].ClusterInfo[job.ClusterIDX].ClusterClientSet, job, monitor.DataCenterInfo[job.DataCenterIDX].ClusterInfo[job.ClusterIDX].NodeInfo[job.NodeIDX].NodeID, NAMESPACE)
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
	log.Println("INFO: Start PersistentPredictor")

	for {
		monitor.GetMetric(1)
		// 对AssignedFailedJob(即ScheduledJob）进行重试，若还是失败，重新放回originJob
		log.Println("INFO: Start ReAssignjob")
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
		log.Println("INFO: Start monitor AssignedJob")
		for idx, ele := range monitor.JobPool.AssignedJob {
			joblist, _ := JobList(monitor.DataCenterInfo[ele.DataCenterIDX].ClusterInfo[ele.ClusterIDX].ClusterClientSet, NAMESPACE)
			for _, job := range joblist.Items {
				if job.Name == ele.Batchv1Job.Name {
					switch {
					case job.Status.Succeeded == 1:
						flag = true
						log.Printf("INFO: AssignedJob finished, %v %v %v %v %v, runtime %v", ele.ID, ele.DataCenterIDX, ele.ClusterIDX, ele.NodeIDX, ele.CardIDX, time.Since(job.Status.StartTime.Time).Seconds())
					case job.Status.Failed == 1:
						flag = true
						log.Printf("ERROR:AssignedJob failed, %v %v %v %v %v", ele.ID, ele.DataCenterIDX, ele.ClusterIDX, ele.NodeIDX, ele.CardIDX)
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
			job := monitor.JobPool.AssignedJob[finishedJobIdx[i]]
			monitor.GetCardInfoPointerFromJob(job).JobQueue.RemoveJob(job.ID)
			monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob[:finishedJobIdx[i]], monitor.JobPool.AssignedJob[finishedJobIdx[i]+1:]...)
		}

		// 当有任务结束时
		if flag {
			monitor.GetMetric(1)

			// 对ReservedJob进行再分配
			var AssignFailedJobQueue = JobQueue{}
			log.Println("INFO: Start assign ReservedJob")

		overloop:
			for _, job := range monitor.JobPool.ReservedJob {
				for _, j := range monitor.GetCardInfoPointerFromJob(job).JobQueue {
					if j.JobType == "GPU" { // 避免一个卡上存在两个GPU Job
						AssignFailedJobQueue = append(AssignFailedJobQueue, job)
						continue overloop
					}
				}
				if int64(time.Since(job.ReservationStartTime).Seconds()) > job.ReservedTime { // 这个方案下，预留任务上去了可能会遇到显存不够的状态，导致任务失败。解决方案：1. 分配任务前，先检验显存是否够；2. 给Job设定RestartPolicy，定时重启来抢占资源
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
			if len(monitor.JobPool.OriginJob) != 0 {
				log.Println("INFO: ReAssign OriginJob")
				monitor.ScheduleAndAssign()
			}
		}

		if len(monitor.JobPool.OriginJob) == 0 && len(monitor.JobPool.ScheduledJob) == 0 && len(monitor.JobPool.AssignedJob) == 0 {
			return
		}

		log.Println("INFO: AssignedJob:", monitor.JobPool.AssignedJob.GetID())
		log.Println("INFO: Sleeping...")
		time.Sleep(time.Minute)
	}
}

func (monitor *Monitor) DeleteJobFromNode(clientset *kubernetes.Clientset, job *Job, namespace string) bool {
	err := clientset.BatchV1().Jobs(namespace).Delete(context.TODO(), job.Batchv1Job.Name, metav1.DeleteOptions{GracePeriodSeconds: new(int64), PropagationPolicy: ptr.To(metav1.DeletePropagationForeground)})
	if err != nil {
		log.Println("ERROR: DeleteJobFromNode failed!", err)
		return false
	}
	return true
}
