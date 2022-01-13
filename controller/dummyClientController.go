package controller

import (
	"context"
	"fmt"
	clientset "github.com/wm775825/sync-controller/pkg/generated/clientset/versioned"
	informers "github.com/wm775825/sync-controller/pkg/generated/informers/externalversions/serverless/v1alpha1"
	listers "github.com/wm775825/sync-controller/pkg/generated/listers/serverless/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// To simplify experiments, we couple client and controller in one struct.
type dummyClientController struct {
	kubeClientset kubernetes.Interface
	serverlessClientset clientset.Interface
	simageLister listers.SimageLister
	simagesSyced cache.InformerSynced

	testImageTag string
}

func NewDummyClientController(kubeClientset kubernetes.Interface,
	serverlessClientset clientset.Interface,
	simageInformer informers.SimageInformer,
	dummyImageTag string) *dummyClientController {
	return &dummyClientController{
		kubeClientset:			kubeClientset,
		serverlessClientset:	serverlessClientset,
		simageLister:       	simageInformer.Lister(),
		simagesSyced:        	simageInformer.Informer().HasSynced,
		testImageTag:			dummyImageTag,
	}
}

func (c *dummyClientController) Run(stopCh chan struct{}) error {
	defer utilruntime.HandleCrash()

	klog.Info("Starting sync controller")

	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.simagesSyced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	klog.Info("informer caches synced")

	klog.Info("Start sync local images periodically")
	go wait.Until(c.dummySyncLocalImages, 10 * time.Second, stopCh)

	klog.Info("Start listening and serving")
	go wait.Until(c.dummyListenAndServe, 10 * time.Second, stopCh)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	klog.Infof("Receive signal: %v\n", <-signalCh)
	stopCh <- struct{}{}
	klog.Info("Shutting down sync controller")
	return nil
}

func (c *dummyClientController) dummySyncLocalImages() {
	// If we need to update it, we should make sure that this simage exists
	// if not, return
	if simage, err := c.simageLister.Simages(defaultNamespace).Get(c.testImageTag); err == nil {
		newSimage := simage.DeepCopy()
		registries := newSimage.Spec.Registries

		// Random select one existing url and append it into registries list
		// This may need further considering
		rand.Seed(time.Now().UnixNano())
		dummyUrl := registries[rand.Intn(len(registries))]
		newSimage.Spec.Registries = append(newSimage.Spec.Registries, dummyUrl)
		if _, err := c.serverlessClientset.ServerlessV1alpha1().Simages(defaultNamespace).Update(context.TODO(), newSimage, metav1.UpdateOptions{}); err != nil {
			utilruntime.HandleError(err)
			return
		}
		klog.Infof("simageClient update %s\n", c.testImageTag)
	}
}

func (c *dummyClientController) dummyListenAndServe() {
	var ret string
	image, err := c.simageLister.Simages(defaultNamespace).Get(c.testImageTag)
	if err != nil {
		ret = defaultRegistryUrl
	} else {
		rand.Seed(time.Now().UnixNano())
		registries := image.Spec.Registries
		ret = registries[rand.Intn(len(registries))]
	}
	klog.Infof("Server: get registry url %s for %s\n", ret, c.testImageTag)
}
