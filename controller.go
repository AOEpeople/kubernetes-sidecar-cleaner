package main

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller demonstrates how to implement a controller with client-go.
type Controller struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
	callback CleanerCallback
}

// NewController creates a new Controller.
func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller, callback CleanerCallback) *Controller {
	return &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
		callback: callback,
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.checkContainerStatus(key.(string))
	c.handleErr(err, key)
	return true
}

func (c *Controller) checkContainerStatus(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		return nil
	}

	klog.Infof("Checking Pod %s/%s (%s)", obj.(*v1.Pod).Namespace, obj.(*v1.Pod).Name, obj.(*v1.Pod).Status.Phase)
	if obj.(*v1.Pod).Status.Phase == v1.PodFailed || obj.(*v1.Pod).Status.Phase == v1.PodSucceeded {
		return nil
	}

	// Note that you also have to check the uid if you have a local controlled resource, which
	// is dependent on the actual instance, to detect that a Pod was recreated with the same name
	//fmt.Printf("Sync/Add/Update for Pod %s\n", obj.(*v1.Pod).GetName())

	activeCount := 0
	terminatedCount := 0
	errorCount := 0
	for _, containerStatus := range obj.(*v1.Pod).Status.ContainerStatuses {
		if strings.HasPrefix(containerStatus.Name, "istio-") {
			continue
		}

		if containerStatus.State.Waiting != nil || containerStatus.State.Running != nil {
			activeCount = activeCount + 1
		}
		if containerStatus.State.Terminated != nil {
			terminatedCount = terminatedCount + 1
			if containerStatus.State.Terminated.Reason == "Error" && obj.(*v1.Pod).Spec.RestartPolicy != "Never" {
				errorCount = errorCount + 1
			}
		}
	}

	if activeCount != 0 || terminatedCount == 0 || errorCount != 0 {
		return nil
	}

	klog.Infof("removing %s %d %d\n", obj.(*v1.Pod).GetName(), activeCount, terminatedCount)

	return c.callback(obj.(*v1.Pod))

}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		c.queue.Forget(key)
		return
	}

	if c.queue.NumRequeues(key) < 5 {
		klog.Infof("Error syncing pod %v: %v", key, err)
		c.queue.AddRateLimited(key)
		return
	}
	c.queue.Forget(key)
	runtime.HandleError(err)
	klog.Infof("Dropping pod %q out of the queue: %v", key, err)
}

// Run begins watching and syncing.
func (c *Controller) Run(workers int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	klog.Info("Starting Pod controller")

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Info("Stopping Pod controller")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}
