package main

import (
	"flag"
	"github.com/docker/docker/client"
	controller2 "github.com/wm775825/sync-controller/controller"
	clientset "github.com/wm775825/sync-controller/pkg/generated/clientset/versioned"
	informers "github.com/wm775825/sync-controller/pkg/generated/informers/externalversions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"os"
	"path/filepath"
	"time"
)

var (
	kubeconfig string
	masterURL string
	dummy bool
	dummyImageTag string
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig(), "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "https://192.168.1.116:6443", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&dummy, "dummy", false, "Run dummy controller or real controller.")
	flag.StringVar(&dummyImageTag, "tag","openwhisk/action-python-v3.7:1.17.0", "dummy image tag used by dummy controller.")

	klog.InitFlags(nil)

	flag.Parse()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		cfg, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
		if err != nil {
			klog.Fatalf("Error building kubeconfig: %s", err.Error())
		}
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	serverlessClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building serverless clientset: %s", err.Error())
	}
	simageInformerFactory := informers.NewSharedInformerFactory(serverlessClient, time.Minute * 10)

	var controller controller2.Controller
	if !dummy {
		dockerClient, err := client.NewClientWithOpts()
		if err != nil {
			klog.Fatalf("Error building docker client: %s", err.Error())
		}
		controller = controller2.NewController(kubeClient, serverlessClient, dockerClient,
			simageInformerFactory.Serverless().V1alpha1().Simages())
	} else {
		controller = controller2.NewDummyClientController(kubeClient, serverlessClient,
			simageInformerFactory.Serverless().V1alpha1().Simages(), dummyImageTag)
	}

	stopCh := make(chan struct{})
	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go simageInformerFactory.Start(stopCh))
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	simageInformerFactory.Start(stopCh)

	if err = controller.Run(stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
	close(stopCh)
}

func defaultKubeconfig() string {
	fname := os.Getenv("KUBECONFIG")
	if fname != "" {
		return fname
	}
	home, err := os.UserHomeDir()
	if err != nil {
		klog.Warningf("failed to get home directory: %v", err)
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}
