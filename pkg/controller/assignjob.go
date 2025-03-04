package controller

import (
	"context"
	"fmt"
	"log"

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

// func (monitor *Monitor) AssignJob() {
// 	failedJobQueue := []*Job{}
// 	for _, job := range monitor.JobPool.AssignedJob {
// 		if AssignJobWithSystem(job) {
// 			monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob, job)
// 			job.ScheduledTime = time.Now()
// 		} else {
// 			failedJobQueue = append(failedJobQueue, job)
// 		}
// 	}
// 	monitor.JobPool.ScheduledJob = failedJobQueue
// }
