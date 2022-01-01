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
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	controllerAgentName = "serverless-controller"
	unixSock = "/tmp/server.sock"
	registrySubNet = "192.168.1"
	registryPort = "5000"
	defaultNamespace = "wm775825"
	defaultRegistryUrl = "192.168.1.141:5000"
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
		server: &http.Server{},
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

	klog.Info("Start listening and serving")
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

	if err := os.Remove(unixSock); err != nil {
		utilruntime.HandleError(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(c.handleQuery))
	c.server.Handler = mux

	listener, err := net.ListenUnix("unix", &net.UnixAddr{
		Name: unixSock,
		Net: "unix",
	})
	if err != nil {
		utilruntime.HandleError(err)
	}

	if err := c.server.Serve(listener); err != nil {
		utilruntime.HandleError(err)
	}
}

func (c *Controller) handleQuery(w http.ResponseWriter, req *http.Request) {
	registryUrl := c.getRegistryUrlByImage(req.URL.Path[1:])
	w.WriteHeader(200)
	_, _ = w.Write([]byte(registryUrl))
}

func (c *Controller) getRegistryUrlByImage(imageId string) string {
	image, err := c.simageLister.Simages(defaultNamespace).Get(imageId)
	if err != nil {
		// TODO: how to deal with error
		utilruntime.HandleError(err)
		return defaultRegistryUrl
	}

	rand.Seed(time.Now().UnixNano())
	registries := image.Spec.Registries
	return registries[rand.Intn(len(registries))]
}

func (c *Controller) syncLocalImages() {

}

func getLocalIp() string {
	ipAddrs, _ := net.InterfaceAddrs()
	for _, addr := range ipAddrs {
		if strings.Contains(addr.String(), registrySubNet) {
			return strings.Split(addr.String(), "/")[0]
		}
	}
	return ""
}