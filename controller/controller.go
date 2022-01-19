package controller

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
	"github.com/wm775825/sync-controller/utils"
	"io"
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
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	controllerAgentName = "serverless-controller"
	unixSock = "/tmp/server.sock"
	registrySubNet = "192.168.1"
	registryPort = "5000"
	defaultNamespace = "wm775825"
	defaultRegistryUrl = "docker.io"
	defaultUserName = "library"
	legalRunes = "1234567890abcdefghijklmnopqrstuvwxyz-."
)

var (
	filterString = "openwhisk"
)

// TODO: decouple syncer, prefetcher and router
type controller struct {
	kubeClientset kubernetes.Interface
	serverlessClientset clientset.Interface
	simageLister listers.SimageLister
	simagesSyced cache.InformerSynced
	recorder record.EventRecorder
	server *http.Server

	sfunctionLister listers.SfunctionLister
	sfunctionsSyced cache.InformerSynced
	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface

	dockerClient *client.Client
	localhostAddr string
	syncedImagesSet map[string]bool
	shaMismatchSet map[string]bool
}

func NewController(
	kubeClientset kubernetes.Interface,
	serverlessClientset clientset.Interface,
	dockerClient *client.Client,
	simageInformer informers.SimageInformer,
	sfunctionInformer informers.SfunctionInformer) *controller {

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

	controller := &controller{
		kubeClientset:       kubeClientset,
		serverlessClientset: serverlessClientset,
		simageLister:        simageInformer.Lister(),
		simagesSyced:        simageInformer.Informer().HasSynced,
		recorder:            recorder,
		server:              &http.Server{},
		sfunctionLister: 	 sfunctionInformer.Lister(),
		sfunctionsSyced:     sfunctionInformer.Informer().HasSynced,
		workqueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Sfunctions"),
		dockerClient:        dockerClient,
		localhostAddr:       getLocalIp() + ":" + registryPort,
		syncedImagesSet:     map[string]bool{},
		shaMismatchSet:      map[string]bool{},
	}

	sfunctionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueSfunction,
	})

	klog.Infof("Sync controller init.")
	return controller
}

func (c *controller) Run(stopCh chan struct{}, sync bool) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("Starting sync controller")

	klog.Info("Waiting for simage caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.simagesSyced); !ok {
		return fmt.Errorf("failed to wait for simage caches to sync")
	}

	klog.Info("Waiting for sfunction caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.sfunctionsSyced); !ok {
		return fmt.Errorf("failed to wait for sfunction caches to sync")
	}

	klog.Info("informer caches synced")

	if sync {
		klog.Info("Start sync local images periodically")
		go wait.Until(c.syncLocalImages, 10 * time.Second, stopCh)
	}

	klog.Info("Start listening and prefetching images")
	go wait.Until(c.listenAndPrefetch, 10 * time.Minute, stopCh)

	klog.Info("Start listening and serving")
	go wait.Until(c.listenAndServe, 10 * time.Second, stopCh)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
	for {
		s := <-signalCh
		klog.Infof("Receive signal: %v\n", s)
		if s != syscall.SIGHUP {
			break
		}
	}
	stopCh <- struct{}{}
	klog.Info("Shutting down sync controller")
	return nil
}

func (c *controller) listenAndPrefetch() {
	for c.processNextWorkItem() {
	}
}

func (c *controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if requeue, err := c.prefetch(key); err != nil {
			return fmt.Errorf("error prefetching '%s': %s", key, err.Error())
		} else if requeue {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
		} else {
			c.workqueue.Forget(obj)
		}
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *controller) prefetch(key string) (bool, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false, fmt.Errorf("invalid resource key: %s", key)
	}

	sfunction, err := c.sfunctionLister.Sfunctions(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, fmt.Errorf("sfunction '%s' in workqueue no longer exists", key)
		}
		return false, err
	}

	prefetchFunc := sfunction.Spec.PrefetchFunc
	var fetch bool
	switch prefetchFunc {
	case "log":
		nodes, err := c.kubeClientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return true, nil
		}
		n := float64(len(nodes.Items))
		rand.Seed(time.Now().UnixNano())
		fetch = rand.Float64() < math.Log2(n) / n
	default:
		return false, fmt.Errorf("wrong prefetch func %s in sfunction '%s'", prefetchFunc, key)
	}

	if fetch {
		return func() (requeue bool, err error) {
			// TODO: moby part:
			// TODO: 	need to deal with images with "docker.io/" prefix.
			// TODO: 	there is no need to re communicate with self.
			requeue = false
			image := "docker.io/" + sfunction.Spec.Image
			resp, err := c.dockerClient.ImagePull(context.TODO(), image, types.ImagePullOptions{
				All:          true,
				RegistryAuth: "arbitrarycodes",
			})
			if err != nil {
				requeue = true
				err = nil
			}
			defer func(resp io.ReadCloser) {
				err = resp.Close()
				requeue = err != nil
				err = nil
			}(resp)
			return
		}()
	}
	return false, nil
}

func (c *controller) enqueueSfunction(obj interface{}){
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *controller) listenAndServe() {
	defer func() {
		if err := c.server.Shutdown(context.TODO()); err != nil {
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
		Net:  "unix",
	})
	if err != nil {
		utilruntime.HandleError(err)
	}

	if err := c.server.Serve(listener); err != nil {
		utilruntime.HandleError(err)
	}
}

func (c *controller) handleQuery(w http.ResponseWriter, req *http.Request) {
	registryUrl := c.getRegistryUrlByImage(req.URL.Path[1:])
	w.WriteHeader(200)
	_, _ = w.Write([]byte(registryUrl))
}

func (c *controller) getRegistryUrlByImage(imageTag string) string {
	var ret string
	legalImageTag := convertToLegalRunes(imageTag)
	image, err := c.simageLister.Simages(defaultNamespace).Get(legalImageTag)
	if err != nil {
		ret = defaultRegistryUrl
	} else {
		registries := image.Spec.Registries
		ret = getRandomURL(registries)
	}
	klog.Infof("Server: get registry url %s for %s\n", ret, imageTag)
	return ret
}

func (c *controller) syncLocalImages() {
	// images with <none>:<none> tag will not be returned by ImageList()
	// however, images with <image>:<none> tag will be returned
	imageList, err := c.dockerClient.ImageList(context.TODO(), types.ImageListOptions{})
	if err != nil {
		utilruntime.HandleError(err)
	}

	for _, image := range imageList{
		imageId := image.ID
		// We do not sync images with <image>:<none> tag
		for _, tag := range image.RepoTags {
			if strings.HasPrefix(tag, registrySubNet) {
				// We do not sync images with 198.168.1.xxx:5000/<image>:<tag> tags,
				// since these are images with <image>:<tag> tag that refer to the same underlying images,
				// we only need to sync the latter images
				continue
			}

			// for experiments, we only sync images of openwhisk.
			if !strings.Contains(tag, filterString) {
				continue
			}

			if found := c.syncedImagesSet[tag]; !found {
				// images with the same tag <image-name>:latest on different nodes may be different
				if shaMismatch := c.shaMismatchSet[tag]; shaMismatch {
					klog.Errorf("[ERROR] SHA256 mismatch for %s in local registry and etcd\n", tag)
				} else {
					c.doSyncImage(imageId, tag)
				}
			}
		}
	}
}

func (c *controller) doSyncImage(imageId, imageTag string) {
	newTag := c.convertImageTag(imageTag)

	// 1.1 docker retag the image
	if err := c.dockerClient.ImageTag(context.TODO(), imageTag, newTag); err != nil {
		utilruntime.HandleError(err)
		return
	}
	klog.Infof("dockerClient retag %s to %s\n", imageTag, newTag)

	// 1.2 push the image to the local registry
	func () {
		resp, err := c.dockerClient.ImagePush(context.TODO(), newTag, types.ImagePushOptions{
			All: true,
			RegistryAuth: "arbitrarycodes",
		})
		if err != nil {
			utilruntime.HandleError(err)
			return
		}
		if err := utils.DisplayResp(resp); err != nil {
			utilruntime.HandleError(err)
			return
		}
		defer resp.Close()
	}()
	klog.Infof("dockerClient push %s to local registry\n", newTag)

	// 1.3 delete the new tag
	defer func() {
		_, err := c.dockerClient.ImageRemove(context.TODO(), newTag, types.ImageRemoveOptions{})
		if err != nil {
			utilruntime.HandleError(err)
		}
		klog.Infof("dockerClient remove tag %s\n", newTag)
	}()

	// 2. notify the etcd of the image/registry info.
	legalImageTag := convertToLegalRunes(imageTag)
	simage, err := c.simageLister.Simages(defaultNamespace).Get(legalImageTag)
	if err == nil {
		// check the of the image
		if simage.Spec.ImageId != imageId {
			c.shaMismatchSet[imageTag] = true
			klog.Errorf("[Error] The tag %s expected image %s, but got image %s. Manual intervention is needed.",
				imageTag, imageId, simage.Spec.ImageId)
			return
		}

		// 2.1. update the Simage object
		newSimage := simage.DeepCopy()
		registries := newSimage.Spec.Registries
		var flag bool
		for _, registry := range registries {
			if registry.URL == c.localhostAddr {
				flag = true
				break
			}
		}
		if !flag {
			newSimage.Spec.Registries = append(newSimage.Spec.Registries,
				v1alpha1.Registry{URL: c.localhostAddr, Weight: 1})
		}
		if _, err := c.serverlessClientset.ServerlessV1alpha1().Simages(defaultNamespace).Update(context.TODO(), newSimage, metav1.UpdateOptions{}); err != nil {
			utilruntime.HandleError(err)
			return
		}
		klog.Infof("simageClient update %s\n", imageTag)
	} else if errors.IsNotFound(err) {
		// 2.1. create a new Simage object
		newSimage := &v1alpha1.Simage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      legalImageTag,
				Namespace: defaultNamespace,
			},
			Spec: v1alpha1.SimageSpec {
				ImageId: imageId,
				Registries: []v1alpha1.Registry{
					{URL: c.localhostAddr, Weight: 1},
				},
			},
		}
		if _, err := c.serverlessClientset.ServerlessV1alpha1().Simages(defaultNamespace).Create(context.TODO(), newSimage, metav1.CreateOptions{}); err != nil {
			utilruntime.HandleError(err)
			return
		}
		klog.Infof("simageClient create %s\n", imageTag)
	} else {
		utilruntime.HandleError(err)
		return
	}

	// 3. update local image set
	c.syncedImagesSet[imageTag] = true
	klog.Infof("Sync controller sync %s\n", imageTag)
}

func (c *controller) convertImageTag(imageTag string) string {
	// the complete format of image tag: domain/user/image:version
	switch strings.Count(imageTag, "/") {
	case 2:
		// 1. domain/user/image:version
		i := strings.IndexRune(imageTag, '/')
		return c.localhostAddr + "/" + imageTag[i+1:]
	case 1:
		i := strings.IndexRune(imageTag, '/')
		if !strings.ContainsAny(imageTag[:i], ".:") && imageTag[:i] != "localhost" {
			// 2. user/image:version
			return c.localhostAddr + "/" + imageTag
		} else {
			// 3. domain/image:version
			return c.localhostAddr + "/" + defaultUserName + "/" + imageTag[i+1:]
		}
	case 0:
		// 4. image:version
		return c.localhostAddr + "/" + defaultUserName + "/" + imageTag
	default:
		// unreachable
		// <none>:<none> images have been prefiltered
		return ""
	}
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

func convertToLegalRunes(s string) string {
	// We also need to make sure that s both begins and ends with alphanumeric runes.
	// But it is an unlikely condition, so we do not take action.
	var ret string
	for _, r := range s {
		if !strings.ContainsRune(legalRunes, r) {
			ret += fmt.Sprintf("0x%d", r)
		} else {
			ret += string(r)
		}
	}
	return ret
}

func getRandomURL(registries []v1alpha1.Registry) string {
	n := len(registries)
	weights := make([]int, n)
	for i, r := range registries {
		if i == 0 {
			weights[i] = r.Weight
		} else {
			weights[i] = weights[i-1] + r.Weight
		}
	}

	rand.Seed(time.Now().UnixNano())
	randomResult := rand.Intn(weights[n-1])
	return registries[sort.Search(n, func(i int) bool { return weights[i] > randomResult })].URL
}
