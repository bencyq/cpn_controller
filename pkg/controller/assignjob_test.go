package controller

import (
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPrepareJobForNodeUsesHamiUUID(t *testing.T) {
	var monitor Monitor
	monitor.unmarshalJson(getJsonWithFile("example2.json"))

	job := &Job{
		ID:            "unit-job",
		DataCenterIDX: 0,
		ClusterIDX:    0,
		NodeIDX:       1,
		CardIDX:       1,
		GPUMemoryReq:  4096,
		Batchv1Job: batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "unit-job"},
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{
							{
								Name: "worker",
								Env: []corev1.EnvVar{
									{Name: "NVIDIA_VISIBLE_DEVICES", Value: "all"},
									{Name: "CUDA_VISIBLE_DEVICES", Value: "7"},
								},
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceName("k8s.amazonaws.com/vgpu"): *resource.NewQuantity(1, resource.DecimalSI),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := monitor.prepareJobForNode(job, "node200"); err != nil {
		t.Fatalf("prepareJobForNode failed: %v", err)
	}

	container := job.Batchv1Job.Spec.Template.Spec.Containers[0]
	if got := job.Batchv1Job.Spec.Template.Annotations["nvidia.com/use-gpuuuid"]; got != "GPU-5e749a00-5ffb-5731-a714-d031839734e8" {
		t.Fatalf("unexpected gpu uuid annotation: %q", got)
	}
	if got := job.Batchv1Job.Spec.Template.Annotations["hami.io/resource-pool"]; got != "poc" {
		t.Fatalf("unexpected hami resource pool: %q", got)
	}
	if got := job.Batchv1Job.Spec.Template.Spec.SchedulerName; got != "hami-scheduler" {
		t.Fatalf("unexpected scheduler name: %q", got)
	}
	if job.Batchv1Job.Spec.Template.Spec.RuntimeClassName == nil || *job.Batchv1Job.Spec.Template.Spec.RuntimeClassName != "nvidia-legacy" {
		t.Fatalf("unexpected runtime class: %#v", job.Batchv1Job.Spec.Template.Spec.RuntimeClassName)
	}
	if got := job.Batchv1Job.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"]; got != "node200" {
		t.Fatalf("unexpected node selector: %q", got)
	}
	if !job.Batchv1Job.Spec.Template.Spec.HostIPC {
		t.Fatal("expected HostIPC to be true")
	}
	if _, ok := container.Resources.Limits[corev1.ResourceName("k8s.amazonaws.com/vgpu")]; ok {
		t.Fatal("expected aws vgpu limit to be removed")
	}
	limit := container.Resources.Limits[corev1.ResourceName("nvidia.com/gpu")]
	if got := limit.Value(); got != 1 {
		t.Fatalf("unexpected hami gpu limit: %d", got)
	}
	request := container.Resources.Requests[corev1.ResourceName("nvidia.com/gpu")]
	if got := request.Value(); got != 1 {
		t.Fatalf("unexpected hami gpu request: %d", got)
	}
	gpuMemLimit := container.Resources.Limits[corev1.ResourceName("nvidia.com/gpumem")]
	if got := gpuMemLimit.Value(); got != 4096 {
		t.Fatalf("unexpected hami gpumem limit: %d", got)
	}
	gpuMemRequest := container.Resources.Requests[corev1.ResourceName("nvidia.com/gpumem")]
	if got := gpuMemRequest.Value(); got != 4096 {
		t.Fatalf("unexpected hami gpumem request: %d", got)
	}
	for _, env := range container.Env {
		if env.Name == "NVIDIA_VISIBLE_DEVICES" {
			t.Fatal("expected NVIDIA_VISIBLE_DEVICES to be removed")
		}
		if env.Name == "CUDA_VISIBLE_DEVICES" && env.Value != "0" {
			t.Fatalf("unexpected CUDA_VISIBLE_DEVICES: %q", env.Value)
		}
	}
	if len(container.VolumeMounts) != 1 || container.VolumeMounts[0].MountPath != "/tmp/nvidia-mps" {
		t.Fatalf("unexpected volume mounts: %#v", container.VolumeMounts)
	}
	if len(job.Batchv1Job.Spec.Template.Spec.Volumes) != 1 || job.Batchv1Job.Spec.Template.Spec.Volumes[0].HostPath == nil || job.Batchv1Job.Spec.Template.Spec.Volumes[0].HostPath.Path != "/tmp/nvidia-mps" {
		t.Fatalf("unexpected volumes: %#v", job.Batchv1Job.Spec.Template.Spec.Volumes)
	}
}

func TestAssignJobWithinController(t *testing.T) {
	monitor := NewMonitor()
	testJob := monitor.JobPool.OriginJob[1]
	testJob.DataCenterIDX, testJob.ClusterIDX, testJob.NodeIDX, testJob.CardIDX = 0, 0, 1, 0
	monitor.AssignJobWithinController(testJob)
}

func TestAssignJobToNode(t *testing.T) {
	JsonUrl = "example2.json"
	monitor := NewMonitor()
	monitor.ReadModelBaseline()
	job := monitor.JobPool.OriginJob[1]
	job.DataCenterIDX, job.ClusterIDX, job.NodeIDX, job.CardIDX = 0, 0, 1, 0
	monitor.JobAnalyze(job)
	job2 := monitor.JobPool.OriginJob[2]
	job2.DataCenterIDX, job2.ClusterIDX, job2.NodeIDX, job2.CardIDX = 0, 0, 1, 0
	monitor.JobAnalyze(job2)
	monitor.AssignJobToNode(monitor.DataCenterInfo[0].ClusterInfo[0].ClusterClientSet, job, "node200", NAMESPACE)
	monitor.AssignJobToNode(monitor.DataCenterInfo[0].ClusterInfo[0].ClusterClientSet, job2, "node200", NAMESPACE)
}

func TestDeleteJobFromNode(t *testing.T) {
	NAMESPACE = `fifo`
	monitor := NewMonitor()
	job := monitor.JobPool.OriginJob[6]
	monitor.DeleteJobFromNode(monitor.DataCenterInfo[0].ClusterInfo[0].ClusterClientSet, job, NAMESPACE)
}
