package controller

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StrategyAllocator func(*Monitor, *Job) bool

type StrategyOptions struct {
	Name                 string
	Namespace            string
	SubmissionInterval   time.Duration
	RetryInterval        time.Duration
	MaxSubmissions       int
	StopAfterSubmissions bool
	Context              context.Context
}

type JobPodStatus struct {
	PodName    string
	Phase      corev1.PodPhase
	NodeName   string
	Reason     string
	Message    string
	Containers []corev1.ContainerStatus
}

func normalizeStrategyOptions(opts StrategyOptions) StrategyOptions {
	if opts.Name == "" {
		opts.Name = "strategy"
	}
	if opts.Namespace == "" {
		opts.Namespace = NAMESPACE
	}
	if opts.SubmissionInterval <= 0 {
		opts.SubmissionInterval = time.Minute
	}
	if opts.RetryInterval <= 0 {
		opts.RetryInterval = 15 * time.Second
	}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	return opts
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func sanitizeJobName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "job"
	}

	var builder strings.Builder
	lastDash := false
	for _, ch := range name {
		switch {
		case ch >= 'a' && ch <= 'z':
			builder.WriteRune(ch)
			lastDash = false
		case ch >= '0' && ch <= '9':
			builder.WriteRune(ch)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}

	sanitized := strings.Trim(builder.String(), "-")
	if sanitized == "" {
		sanitized = "job"
	}
	if len(sanitized) > 63 {
		sanitized = strings.Trim(sanitized[:63], "-")
	}
	if sanitized == "" {
		sanitized = "job"
	}
	return sanitized
}

func (monitor *Monitor) PrefixJobNames(prefix string) {
	prefix = sanitizeJobName(prefix)
	for _, job := range monitor.JobPool.OriginJob {
		if job == nil {
			continue
		}
		name := sanitizeJobName(prefix + "-" + job.Batchv1Job.Name)
		job.Batchv1Job.Name = name
		job.Batchv1Job.ObjectMeta.Name = name
		job.ID = name
	}
}

func (monitor *Monitor) AssignJobInNamespace(job *Job, namespace string) bool {
	if job.GPUMemoryReq > monitor.GetCardInfoPointerFromJob(job).GPU_MEMORY_FREE {
		log.Printf("DEBUG: Job %v GPUMemoryReq not satisfied, %v %v %v %v", job.ID, job.DataCenterIDX, job.ClusterIDX, job.NodeIDX, job.CardIDX)
		return false
	}

	node := monitor.DataCenterInfo[job.DataCenterIDX].ClusterInfo[job.ClusterIDX].NodeInfo[job.NodeIDX]
	if monitor.AssignJobToNode(monitor.DataCenterInfo[job.DataCenterIDX].ClusterInfo[job.ClusterIDX].ClusterClientSet, job, node.NodeID, namespace) {
		log.Printf("INFO: Job %v assigned, %v %v %v %v", job.ID, job.DataCenterIDX, job.ClusterIDX, job.NodeIDX, job.CardIDX)
		return true
	}
	return false
}

func jobQueueContains(queue JobQueue, jobID string) bool {
	for _, job := range queue {
		if job != nil && job.ID == jobID {
			return true
		}
	}
	return false
}

func (monitor *Monitor) RecordAssignedJob(job *Job) {
	job.AssignedTime = time.Now()
	monitor.JobPool.AssignedJob = append(monitor.JobPool.AssignedJob, job)

	cardInfo := monitor.GetCardInfoPointerFromJob(job)
	if !jobQueueContains(cardInfo.JobQueue, job.ID) {
		cardInfo.JobQueue = append(cardInfo.JobQueue, job)
	}
	cardInfo.GPU_MEMORY_USED += job.GPUMemoryReq
	cardInfo.GPU_MEMORY_FREE -= job.GPUMemoryReq
	if cardInfo.GPU_MEMORY_FREE < 0 {
		cardInfo.GPU_MEMORY_FREE = 0
	}
}

func (monitor *Monitor) TryAssignNextJob(opts StrategyOptions, allocator StrategyAllocator) (*Job, bool) {
	monitor.GetMetric(2)

	for idx, job := range monitor.JobPool.OriginJob {
		monitor.JobAnalyze(job)
		if !allocator(monitor, job) {
			continue
		}

		job.Batchv1Job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
		if !monitor.AssignJobInNamespace(job, opts.Namespace) {
			continue
		}

		monitor.RecordAssignedJob(job)
		monitor.JobPool.OriginJob = append(monitor.JobPool.OriginJob[:idx], monitor.JobPool.OriginJob[idx+1:]...)
		log.Printf("INFO: %s submitted job %v", opts.Name, job.ID)
		return job, true
	}

	return nil, false
}

func (monitor *Monitor) MonitorAssignedJobsOnce(strategyName string, namespace string) bool {
	finishedJobIdx := make([]int, 0)
	changed := false

	for idx, ele := range monitor.JobPool.AssignedJob {
		joblist, err := JobList(
			monitor.DataCenterInfo[ele.DataCenterIDX].ClusterInfo[ele.ClusterIDX].ClusterClientSet,
			namespace,
		)
		if err != nil {
			log.Printf("ERROR: %s job list failed for %v: %v", strategyName, ele.ID, err)
			continue
		}

		for _, job := range joblist.Items {
			if job.Name != ele.Batchv1Job.Name {
				continue
			}

			switch {
			case job.Status.Succeeded == 1:
				log.Printf(
					"INFO: %s assigned job finished, %v %v %v %v %v, runtime %v",
					strategyName,
					ele.ID,
					ele.DataCenterIDX,
					ele.ClusterIDX,
					ele.NodeIDX,
					ele.CardIDX,
					time.Since(job.Status.StartTime.Time).Seconds(),
				)
			case job.Status.Failed == 1:
				monitor.DeleteJobFromNode(monitor.GetClusterInfoPointerFromJob(ele).ClusterClientSet, ele, namespace)
				monitor.JobPool.OriginJob = append(monitor.JobPool.OriginJob, ele)
				log.Printf(
					"ERROR: %s assigned job failed, job deleted %v %v %v %v %v",
					strategyName,
					ele.ID,
					ele.DataCenterIDX,
					ele.ClusterIDX,
					ele.NodeIDX,
					ele.CardIDX,
				)
			case job.Status.Active == 1:
				continue
			default:
				continue
			}

			monitor.GetCardInfoPointerFromJob(ele).JobQueue.RemoveJob(ele.ID)
			finishedJobIdx = append(finishedJobIdx, idx)
			changed = true
			break
		}
	}

	for i := len(finishedJobIdx) - 1; i >= 0; i-- {
		monitor.JobPool.AssignedJob = append(
			monitor.JobPool.AssignedJob[:finishedJobIdx[i]],
			monitor.JobPool.AssignedJob[finishedJobIdx[i]+1:]...,
		)
	}

	return changed
}

func (monitor *Monitor) RunStrategy(opts StrategyOptions, allocator StrategyAllocator) {
	opts = normalizeStrategyOptions(opts)
	submittedCount := 0

	for {
		select {
		case <-opts.Context.Done():
			log.Printf("INFO: %s stopped: %v", opts.Name, opts.Context.Err())
			return
		default:
		}

		changed := monitor.MonitorAssignedJobsOnce(opts.Name, opts.Namespace)
		if len(monitor.JobPool.OriginJob) == 0 && len(monitor.JobPool.AssignedJob) == 0 {
			return
		}

		submitted := false
		if opts.MaxSubmissions == 0 || submittedCount < opts.MaxSubmissions {
			if _, ok := monitor.TryAssignNextJob(opts, allocator); ok {
				submitted = true
				submittedCount++
			}
		}

		if submitted {
			if opts.StopAfterSubmissions && opts.MaxSubmissions > 0 && submittedCount >= opts.MaxSubmissions {
				return
			}
			log.Printf("INFO: %s sleeping %v for next job arrival", opts.Name, opts.SubmissionInterval)
			if !sleepWithContext(opts.Context, opts.SubmissionInterval) {
				return
			}
			continue
		}

		if opts.StopAfterSubmissions && opts.MaxSubmissions > 0 && submittedCount >= opts.MaxSubmissions {
			return
		}

		if !changed {
			log.Printf("INFO: %s no submission, retry after %v", opts.Name, opts.RetryInterval)
			if !sleepWithContext(opts.Context, opts.RetryInterval) {
				return
			}
		}
	}
}

func (monitor *Monitor) WaitForJobPodNotPending(job *Job, namespace string, timeout time.Duration) (*JobPodStatus, error) {
	clientset := monitor.GetClusterInfoPointerFromJob(job).ClusterClientSet
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		podList, err := clientset.CoreV1().Pods(namespace).List(
			context.TODO(),
			metav1.ListOptions{LabelSelector: "job-name=" + job.Batchv1Job.Name},
		)
		if err != nil {
			return nil, fmt.Errorf("list pods for job %s: %w", job.ID, err)
		}

		for _, pod := range podList.Items {
			status := &JobPodStatus{
				PodName:    pod.Name,
				Phase:      pod.Status.Phase,
				NodeName:   pod.Spec.NodeName,
				Reason:     pod.Status.Reason,
				Message:    pod.Status.Message,
				Containers: pod.Status.ContainerStatuses,
			}
			if pod.Status.Phase != corev1.PodPending {
				return status, nil
			}
		}

		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("job %s pod stayed pending for more than %v", job.ID, timeout)
}
