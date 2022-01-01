package main

import (
	"context"
	"fmt"
	clientset "github.com/wm775825/sync-controller/pkg/generated/clientset/versioned"
	serverlessScheme "github.com/wm775825/sync-controller/pkg/generated/clientset/versioned/scheme"
	informers "github.com/wm775825/sync-controller/pkg/generated/informers/externalversions/serverless/v1alpha1"
	listers "github.com/wm775825/sync-controller/pkg/generated/listers/serverless/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"net"
	"net/http"
	"time"
)

const (
	controllerAgentName = "serverless-controller"
	listenAddr = "/tmp/server.sock"
)

type Controller struct {
	kubeClientset kubernetes.Interface
	serverlessClientset clientset.Interface
	simageLister listers.SimageLister
	simagesSyced cache.InformerSynced
	recorder record.EventRecorder
	server *http.Server
}

func NewController(
	kubeClientset kubernetes.Interface,
	serverlessClientset clientset.Interface,
	simageInformer informers.SimageInformer) *Controller {

	// Create event broadcaster
	// Add serverless-controller types to the default Kubernetes Scheme so Events can be
	// logged for serverless-controller types.
	utilruntime.Must(serverlessScheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: kubeClientset.CoreV1().Events(""),
	})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{
		Component: controllerAgentName,
	})

	controller := &Controller{
		kubeClientset: kubeClientset,
		serverlessClientset: serverlessClientset,
		simageLister: simageInformer.Lister(),
		simagesSyced: simageInformer.Informer().HasSynced,
		recorder: recorder,
		server: &http.Server{
			Addr: listenAddr,
		},
	}

	klog.Infof("Sync controller init.")
	return controller
}

func (c *Controller) Run(stopCh <-chan struct{}) error {
	klog.Info("Starting sync controller")

	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.simagesSyced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Start sync local images periodically")
	go wait.Until(c.syncLocalImages, 10 * time.Second, stopCh)

	klog.Info("Start listening at")
	go wait.Until(c.listenAndServe, 10 * time.Second, stopCh)

	<-stopCh
	klog.Info("Shutting down sync controller")
	return nil
}

func (c *Controller) listenAndServe() {
	defer func() {
		if err := c.server.Shutdown(context.Background()); err != nil {
			utilruntime.HandleError(err)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(c.handleQuery))
	c.server.Handler = mux

	listener, err := net.Listen("unix", c.server.Addr)
	if err != nil {
		utilruntime.HandleError(err)
	}

	if err := c.server.Serve(listener); err != nil {
		utilruntime.HandleError(err)
	}
}

func (c *Controller) handleQuery(w http.ResponseWriter, req *http.Request) {
	// TODO: handle query
}

func (c *Controller) syncLocalImages() {

}