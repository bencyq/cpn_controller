package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "k8s.io/apimachinery/pkg/util/yaml"

	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

func upsertEnvVar(envs []corev1.EnvVar, name string, value string) []corev1.EnvVar {
	for idx := range envs {
		if envs[idx].Name == name {
			envs[idx].Value = value
			return envs
		}
	}
	return append(envs, corev1.EnvVar{Name: name, Value: value})
}

func removeEnvVar(envs []corev1.EnvVar, name string) []corev1.EnvVar {
	filtered := envs[:0]
	for _, env := range envs {
		if env.Name == name {
			continue
		}
		filtered = append(filtered, env)
	}
	return filtered
}

func ensureVolumeMount(container *corev1.Container, name string, mountPath string) {
	for idx := range container.VolumeMounts {
		if container.VolumeMounts[idx].Name == name {
			container.VolumeMounts[idx].MountPath = mountPath
			return
		}
	}
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{Name: name, MountPath: mountPath})
}

func ensureHostPathVolume(volumes []corev1.Volume, name string, path string) []corev1.Volume {
	for idx := range volumes {
		if volumes[idx].Name == name {
			volumes[idx].VolumeSource = corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: path}}
			return volumes
		}
	}
	return append(volumes, corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: path}}})
}

func (monitor *Monitor) prepareJobForNode(job *Job, nodeName string) error {
	cardUUID := monitor.GetCardUUIDFromJob(job)
	if cardUUID == "" {
		return fmt.Errorf("job %s selected card has empty UUID", job.ID)
	}

	for i := range job.Batchv1Job.Spec.Template.Spec.Containers {
		container := &job.Batchv1Job.Spec.Template.Spec.Containers[i]
		container.Env = removeEnvVar(container.Env, "NVIDIA_VISIBLE_DEVICES")
		container.Env = upsertEnvVar(container.Env, "CUDA_VISIBLE_DEVICES", "0")

		if container.Resources.Limits == nil {
			container.Resources.Limits = make(corev1.ResourceList)
		}
		delete(container.Resources.Limits, corev1.ResourceName("k8s.amazonaws.com/vgpu"))
		container.Resources.Limits[corev1.ResourceName("nvidia.com/gpu")] = *resource.NewQuantity(int64(1), resource.DecimalSI)
		if job.GPUMemoryReq > 0 {
			container.Resources.Limits[corev1.ResourceName("nvidia.com/gpumem")] = *resource.NewQuantity(job.GPUMemoryReq, resource.DecimalSI)
		}

		if container.Resources.Requests == nil {
			container.Resources.Requests = make(corev1.ResourceList)
		}
		delete(container.Resources.Requests, corev1.ResourceName("k8s.amazonaws.com/vgpu"))
		container.Resources.Requests[corev1.ResourceName("nvidia.com/gpu")] = *resource.NewQuantity(int64(1), resource.DecimalSI)
		if job.GPUMemoryReq > 0 {
			container.Resources.Requests[corev1.ResourceName("nvidia.com/gpumem")] = *resource.NewQuantity(job.GPUMemoryReq, resource.DecimalSI)
		}

		ensureVolumeMount(container, "nvidia-mps", "/tmp/nvidia-mps")
	}

	if job.Batchv1Job.Spec.Template.Spec.NodeSelector == nil {
		job.Batchv1Job.Spec.Template.Spec.NodeSelector = make(map[string]string)
	}
	job.Batchv1Job.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"] = nodeName

	if job.Batchv1Job.Spec.Template.Annotations == nil {
		job.Batchv1Job.Spec.Template.Annotations = make(map[string]string)
	}
	job.Batchv1Job.Spec.Template.Annotations["hami.io/resource-pool"] = HamiResourcePool
	job.Batchv1Job.Spec.Template.Annotations["nvidia.com/use-gpuuuid"] = cardUUID
	delete(job.Batchv1Job.Spec.Template.Annotations, "nvidia.com/nouse-gpuuuid")

	job.Batchv1Job.Spec.Template.Spec.SchedulerName = HamiSchedulerName
	job.Batchv1Job.Spec.Template.Spec.RuntimeClassName = ptr.To(HamiRuntimeClassName)
	job.Batchv1Job.Spec.Template.Spec.HostIPC = true
	job.Batchv1Job.Spec.Template.Spec.Volumes = ensureHostPathVolume(job.Batchv1Job.Spec.Template.Spec.Volumes, "nvidia-mps", "/tmp/nvidia-mps")

	return nil
}

func (monitor *Monitor) AssignJobToNode(clientset *kubernetes.Clientset, job *Job, nodeName string, namespace string) bool {
	if err := monitor.prepareJobForNode(job, nodeName); err != nil {
		log.Println("ERROR: prepareJobForNode failed!", err)
		return false
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

			for _, job := range monitor.JobPool.ReservedJob {
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
