package rr

import (
	"context"
	"cpn-controller/pkg/controller"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const (
	defaultSubmissionInterval = time.Minute
	defaultRetryInterval      = time.Second * 10
)

var NAMESPACE = `rr`

type gpuSlot struct {
	dataCenterIdx int
	clusterIdx    int
	nodeIdx       int
	cardIdx       int
}

type roundRobinState struct {
	slots    []gpuSlot
	nextSlot int
}

func normalizeOptions(opts controller.StrategyOptions) controller.StrategyOptions {
	if opts.Name == "" {
		opts.Name = "rr"
	}
	if opts.Namespace == "" {
		opts.Namespace = NAMESPACE
	}
	if opts.SubmissionInterval <= 0 {
		opts.SubmissionInterval = defaultSubmissionInterval
	}
	if opts.RetryInterval <= 0 {
		opts.RetryInterval = defaultRetryInterval
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

func collectGPUSlots(monitor *controller.Monitor) []gpuSlot {
	slots := make([]gpuSlot, 0)
	for dc, dataCenterInfo := range monitor.DataCenterInfo {
		for cl, clusterInfo := range dataCenterInfo.ClusterInfo {
			for n, nodeInfo := range clusterInfo.NodeInfo {
				for c := range nodeInfo.CardInfo {
					slots = append(slots, gpuSlot{
						dataCenterIdx: dc,
						clusterIdx:    cl,
						nodeIdx:       n,
						cardIdx:       c,
					})
				}
			}
		}
	}
	return slots
}

func (state *roundRobinState) next() (gpuSlot, bool) {
	if len(state.slots) == 0 {
		return gpuSlot{}, false
	}

	slot := state.slots[state.nextSlot%len(state.slots)]
	state.nextSlot = (state.nextSlot + 1) % len(state.slots)
	return slot, true
}

func assignNextJob(monitor *controller.Monitor, namespace string, state *roundRobinState) (*controller.Job, bool) {
	if len(monitor.JobPool.OriginJob) == 0 {
		return nil, false
	}

	slot, ok := state.next()
	if !ok {
		return nil, false
	}

	job := monitor.JobPool.OriginJob[0]
	job.DataCenterIDX = slot.dataCenterIdx
	job.ClusterIDX = slot.clusterIdx
	job.NodeIDX = slot.nodeIdx
	job.CardIDX = slot.cardIdx
	job.Batchv1Job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure

	cluster := monitor.DataCenterInfo[slot.dataCenterIdx].ClusterInfo[slot.clusterIdx]
	node := cluster.NodeInfo[slot.nodeIdx]
	if !monitor.AssignJobToNode(cluster.ClusterClientSet, job, node.NodeID, namespace) {
		log.Printf(
			"ERROR: rr assign failed for job %v on %v %v %v %v",
			job.ID,
			job.DataCenterIDX,
			job.ClusterIDX,
			job.NodeIDX,
			job.CardIDX,
		)
		return nil, false
	}

	monitor.RecordAssignedJob(job)
	monitor.JobPool.OriginJob = monitor.JobPool.OriginJob[1:]
	log.Printf(
		"INFO: rr submitted job %v to %v %v %v %v",
		job.ID,
		job.DataCenterIDX,
		job.ClusterIDX,
		job.NodeIDX,
		job.CardIDX,
	)
	return job, true
}

func Run(monitor *controller.Monitor, opts controller.StrategyOptions) {
	opts = normalizeOptions(opts)
	state := &roundRobinState{slots: collectGPUSlots(monitor)}
	if len(state.slots) == 0 {
		log.Printf("WARN: %s found no GPU slots", opts.Name)
		return
	}

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
			if _, ok := assignNextJob(monitor, opts.Namespace, state); ok {
				submitted = true
				submittedCount++
			}
		}

		if submitted {
			if opts.StopAfterSubmissions && opts.MaxSubmissions > 0 && submittedCount >= opts.MaxSubmissions {
				return
			}
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

func RRSchedule(monitor *controller.Monitor) {
	Run(monitor, controller.StrategyOptions{
		Name:      "rr",
		Namespace: NAMESPACE,
	})
}

func MonitorAssignedJob(monitor *controller.Monitor) {
	RRSchedule(monitor)
}
