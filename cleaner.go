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
	return isOwnedByJob(pod) && isRunning(pod) && hasIstioSidecar(pod) && !hasEmbeddedSidecarCleanup(pod)
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
		err = exec.Stream(remotecommand.StreamOptions{
			Stdout: buf,
			Stderr: errBuf,
		})
		if err != nil {
			return fmt.Errorf("%w Failed executing on %v/%v\n%s\n%s", err, pod.Namespace, pod.Name, buf.String(), errBuf.String())
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

func isRunning(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodRunning
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
	if val, ok := pod.Annotations["mesh.bare.id/sidecar-cleanup"]; ok {
		return val == "embedded"
	}
	return false
}
