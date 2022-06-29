package main

import (
	"bytes"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
)

type Cleaner struct {
	restCfg    *rest.Config
	restClient rest.Interface
}

type CleanerCallback func(*v1.Pod) error

func NewCleaner(config *rest.Config, client rest.Interface) *Cleaner {
	return &Cleaner{restCfg: config, restClient: client}
}

//CanProcess checks a pod whether a cleanup process for the istio-proxy is required or not
func (c *Cleaner) CanProcess(pod *v1.Pod) bool {
	condition := isOwnedByJob(pod) && hasIstioSidecar(pod) && !hasEmbeddedSidecarCleanup(pod) && (isRunningOrPending(pod) || isPendingContainerError(pod))
	klog.Infof("CanProcess for Pod with name %s returns %v", pod.GetName(), condition)
	return condition
}

//ProcessCallback is triggered once a pod has only the istio related container left running
func (c *Cleaner) ProcessCallback() CleanerCallback {
	return func(pod *v1.Pod) error {
		klog.Infof("removing %s", pod.GetName())

		buf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		request := c.restClient.
			Post().
			Namespace(pod.Namespace).
			Resource("pods").
			Name(pod.Name).
			SubResource("exec").
			VersionedParams(&v1.PodExecOptions{
				Container: IstioProxy,
				Command:   []string{"/bin/sh", "-c", "curl --max-time 2 -s -f -XPOST http://127.0.0.1:15000/quitquitquit"},
				Stdin:     false,
				Stdout:    true,
				Stderr:    true,
				TTY:       true,
			}, scheme.ParameterCodec)
		exec, err := remotecommand.NewSPDYExecutor(c.restCfg, "POST", request.URL())
		if err != nil {
			return fmt.Errorf("%w failed running the exec on %v/%v\n%s\n%s", err, pod.Namespace, pod.Name, buf.String(), errBuf.String())
		}
		err = exec.Stream(remotecommand.StreamOptions{
			Stdout: buf,
			Stderr: errBuf,
		})
		if err != nil {
			return fmt.Errorf("%w failed executing on %v/%v\n%s\n%s", err, pod.Namespace, pod.Name, buf.String(), errBuf.String())
		}
		return nil
	}
}

const IstioProxy = "istio-proxy"

func isOwnedByJob(pod *v1.Pod) bool {
	for _, owner := range pod.GetOwnerReferences() {
		if owner.Kind == "Job" {
			return true
		}
	}
	return false
}

func isRunningOrPending(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodRunning || pod.Status.Phase == v1.PodPending
}

func isPendingContainerError(pod *v1.Pod) bool {
	if pod.Status.Phase == v1.PodPending {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Waiting != nil {
				if containerStatus.State.Waiting.Reason != "ContainerCreating" {
					klog.Infof("Pending Pod with ContainerStatus %s in container with image %s",
						containerStatus.State.Waiting.Reason,
						containerStatus.Image)
					return true
				}
			}
		}
	}
	return false
}

func hasIstioSidecar(pod *v1.Pod) bool {
	sidecarFound := false
	for _, container := range pod.Spec.Containers {
		if container.Name == "istio-quitquitquit" { // problem solved already
			return false
		}
		if container.Name == IstioProxy {
			sidecarFound = true
		}
	}
	return sidecarFound
}

func hasEmbeddedSidecarCleanup(pod *v1.Pod) bool {
	if val, ok := pod.Annotations["aoe.com/sidecar-cleaner"]; ok {
		return val == "embedded"
	}
	return false
}
