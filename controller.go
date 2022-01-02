package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/wm775825/sync-controller/pkg/apis/serverless/v1alpha1"
	clientset "github.com/wm775825/sync-controller/pkg/generated/clientset/versioned"
	serverlessScheme "github.com/wm775825/sync-controller/pkg/generated/clientset/versioned/scheme"
	informers "github.com/wm775825/sync-controller/pkg/generated/informers/externalversions/serverless/v1alpha1"
	listers "github.com/wm775825/sync-controller/pkg/generated/listers/serverless/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	dockerClient *client.Client
	localhostAddr string
	syncedImagesSet map[string]bool
}

func NewController(
	kubeClientset kubernetes.Interface,
	serverlessClientset clientset.Interface,
	dockerClient *client.Client,
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
		dockerClient: dockerClient,
		localhostAddr: getLocalIp() + ":" + registryPort,
		syncedImagesSet: map[string]bool{},
	}

	klog.Infof("Sync controller init.")
	return controller
}

func (c *Controller) Run(stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()

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
	imageList, err := c.dockerClient.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		utilruntime.HandleError(err)
	}

	for _, image := range imageList{
		imageId := image.ID
		for _, tag := range image.RepoTags {
			if found := c.syncedImagesSet[tag]; !found {
				c.doSyncImage(imageId, tag)
			}
		}
	}
}

func (c *Controller) doSyncImage(imageId, imageTag string) {
	// TODO: user imageId to verify or not?
	newTag := convertImageTag(imageTag)

	// push the image to the local registry
	if _, err := c.dockerClient.ImagePush(context.Background(), newTag, types.ImagePushOptions{}); err != nil {
		utilruntime.HandleError(err)
		return
	}

	// notify the etcd of the image/registry info.
	simage, err := c.simageLister.Simages(defaultNamespace).Get(imageTag)
	if err == nil {
		// update the Simage object
		newSimage := simage.DeepCopy()
		newSimage.Spec.Registries = append(newSimage.Spec.Registries, c.localhostAddr)
		if _, err := c.serverlessClientset.ServerlessV1alpha1().Simages(defaultNamespace).Update(context.Background(), newSimage, metav1.UpdateOptions{}); err != nil {
			utilruntime.HandleError(err)
			return
		}
	} else if errors.IsNotFound(err) {
		// create a new Simage object
		newSimage := &v1alpha1.Simage{
			ObjectMeta: metav1.ObjectMeta{
				Name: imageTag,
				Namespace: defaultNamespace,
			},
			Spec: v1alpha1.SimageSpec {
				Registries: []string{c.localhostAddr},
			},
		}
		if _, err := c.serverlessClientset.ServerlessV1alpha1().Simages(defaultNamespace).Create(context.Background(), newSimage, metav1.CreateOptions{}); err != nil {
			utilruntime.HandleError(err)
			return
		}
	} else {
		utilruntime.HandleError(err)
		return
	}

	// update local image set
	c.syncedImagesSet[imageTag] = true
}

func convertImageTag(imageTag string) string {
	// TODO: how to convert the tag of image
	return ""
}

func getLocalIp() string {
	ipAddrs, _ := net.InterfaceAddrs()
	for _, addr := range ipAddrs {
		if strings.Contains(addr.String(), registrySubNet) {
			return strings.Split(addr.String(), "/")[0]
		}
	}
	// never reachable by default
	return ""
}
