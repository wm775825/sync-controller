package main

import (
	"flag"
	"github.com/docker/docker/client"
	clientset "github.com/wm775825/sync-controller/pkg/generated/clientset/versioned"
	informers "github.com/wm775825/sync-controller/pkg/generated/informers/externalversions"
	"k8s.io/apimachinery/pkg/util/wait"
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
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig(), "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

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

	dockerClient, err := client.NewClientWithOpts()
	if err != nil {
		klog.Fatalf("Error building docker client: %s", err.Error())
	}

	controller := NewController(kubeClient, serverlessClient, dockerClient,
		simageInformerFactory.Serverless().V1alpha1().Simages())

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go simageInformerFactory.Start(stopCh))
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	simageInformerFactory.Start(wait.NeverStop)

	if err = controller.Run(wait.NeverStop); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
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
