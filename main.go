package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"

	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"

	"k8s.io/client-go/rest"
)

type PodMetricDatum struct {
	Timestamp time.Time
	Window    time.Duration
	Value     int64
}

type PodMetrics map[string]PodMetricDatum

type NodeMetricDatum struct {
	Timestamp time.Time
	Window    time.Duration
	Value     int64
}

type NodeMetrics map[string]NodeMetricDatum

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()
	flag.Set("v", os.Getenv("VERBOSE"))
	flag.Parse()

	logger := klogr.NewWithOptions()
	ctx := logr.NewContext(context.Background(), logger)

	var defaultNamespace string
	var config *rest.Config
	var err error

	// unless running with special envvar...
	if _, ok := os.LookupEnv("RESOURCE_AUTOSCALER_TESTING_MODE"); ok {
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
		defaultNamespace = "resource-autoscaler"
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}

		// Get the namespace we are deployed into... that shouldn't be something configured.
		ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			panic(err.Error())
		}
		defaultNamespace = string(ns)

	}

	logger.V(3).Info("Running...", "namespace", defaultNamespace)

	// panics if fails
	clientset := kubernetes.NewForConfigOrDie(config)

	for {
		percentage, err := getResourceUtilization(ctx, resourceCPU, config)
		if err != nil {
			panic(err.Error())
		}

		logger.Info("Percent utilization", "resource", corev1.ResourceCPU.String(), "usage_pct", fmt.Sprintf("%.2f", percentage*100.0))

		// Configuration for the scaler to come from a configmap... can still be JSON payload and therefore be a path in the system
		// I.e. don't do this >>
		// cm, err := clientset.CoreV1().ConfigMaps(myNamespace).Get(ctx, "resource-scaler", metav1.GetOptions{})

		// if errors.IsNotFound(err) {
		// 	fmt.Printf("CM cm-map-name not found\n")
		// } else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		// 	fmt.Printf("Error getting CM %v\n", statusError.ErrStatus.Message)
		// } else if err != nil {
		// 	panic(err.Error())
		// } else {
		// 	fmt.Printf("Yay!!! We found the CM: %+v\n", cm)
		// }

		n, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Error(err, "Error getting nodes")
		}

		logger.V(2).Info("Node count", "nodeCount", len(n.Items))

		if _, ok := os.LookupEnv("RESOURCE_AUTOSCALER_TESTING_MODE"); ok {
			// Running locally... don't sleep
			logger.V(1).Info("Running locally, exiting")
			break
		}

		time.Sleep(10 * time.Second)
	}
}

func getResourceUtilization(ctx context.Context, resource corev1.ResourceName, config *rest.Config) (float64, error) {
	logger := logr.FromContext(ctx)
	metricClient := resourceclient.NewForConfigOrDie(config)
	nodeMetrics, err := metricClient.NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error getting NodeMetrics: %v\n", err.Error())
	}
	// podMetrics, err := metricClient.PodMetricses("").List(ctx, metav1.ListOptions{}) // Add the selector into the ListOptions
	// if err != nil {
	// 	fmt.Printf("Error getting PodMetrics: %v\n", err.Error())
	// }

	// pMetrics := getPodMetrics(podMetrics.Items, resource)
	nMetrics := getNodeMetrics(ctx, nodeMetrics.Items, resource)

	var nodeResourceUsage, nodeResourceAvailable int64

	// for k, m := range pMetrics {
	// logger.V(2).Infof("Found metric %v for pod %v: %v\n", resource, k, m.Value) // TODO: Calculate value as rate over window
	// podResourceUtilization += m.Value
	// }
	nodeResourceAvailable = getNodeResourcesAllocatable(ctx, resourceCPU, config)
	for k, m := range nMetrics {
		logger.V(3).Info("Found node metric", "resource", resource.String(), "node", k, "value", m.Value)
		nodeResourceUsage += m.Value
	}
	logger.V(1).Info("Node utilization in millicores", "value", nodeResourceUsage)
	logger.V(1).Info("Node resources allocated (millicores)", "value", nodeResourceAvailable)

	return float64(nodeResourceUsage) / float64(nodeResourceAvailable), nil
}

// resourceNames
type CoreResourceNames = []corev1.ResourceName

func ResourceNames() []corev1.ResourceName {
	return []corev1.ResourceName{
		corev1.ResourceCPU,
		corev1.ResourceEphemeralStorage,
		corev1.ResourceHugePagesPrefix,
		corev1.ResourceMemory,
		corev1.ResourceStorage,
	}
}

var resourceCPU = corev1.ResourceCPU

// func getPodMetrics(rawPodMetrics []v1beta1.PodMetrics, resource corev1.ResourceName) PodMetrics {
// 	res := make(PodMetrics, len(rawPodMetrics))
// 	for _, m := range rawPodMetrics {
// 		podSum := int64(0)
// 		missing := len(m.Containers) == 0
// 		for _, c := range m.Containers {
// 			resValue, found := c.Usage[resource]
// 			if !found {
// 				missing = true
// 				fmt.Printf("missing resource metric %v for %s/%s\n", resource, m.Namespace, m.Name)
// 				break
// 			}
// 			podSum += resValue.MilliValue()
// 		}
// 		if !missing {
// 			res[m.Name] = PodMetricDatum{
// 				Timestamp: m.Timestamp.Time,
// 				Window:    m.Window.Duration,
// 				Value:     podSum,
// 			}
// 		}
// 	}
// 	return res
// }

func getNodeResourcesAllocatable(ctx context.Context, resourceName corev1.ResourceName, config *rest.Config) int64 {
	clientset := kubernetes.NewForConfigOrDie(config)

	n, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	var allocatableResource int64

	for _, node := range n.Items {
		quantity := node.Status.Allocatable[resourceName]
		allocatableResource += quantity.MilliValue()
	}

	return allocatableResource
}

// 	res := make(NodeMetrics, len(rawNodeMetrics))
// 	for _, m := range rawNodeMetrics {
// 		resValue, found := m.Usage[resource]
// 		if !found {
// 			fmt.Printf("Missing resource metric %v for %s/%s\n", resource, m.Namespace, m.Name)
// 			break
// 		}
// 		res[m.Name] = NodeMetricDatum{
// 			Timestamp: m.Timestamp.Time,
// 			Window:    m.Window.Duration,
// 			Value:     resValue.MilliValue(),
// 		}
// 	}
// 	return res
// }

func getNodeMetrics(ctx context.Context, rawNodeMetrics []v1beta1.NodeMetrics, resource corev1.ResourceName) NodeMetrics {
	logger := logr.FromContext(ctx)
	res := make(NodeMetrics, len(rawNodeMetrics))
	for _, m := range rawNodeMetrics {
		resValue, found := m.Usage[resource]
		if !found {
			logger.Info("Missing resource metric", "resourceName", resource.String(), "namespace", m.Namespace, "name", m.Name)
			break
		}
		res[m.Name] = NodeMetricDatum{
			Timestamp: m.Timestamp.Time,
			Window:    m.Window.Duration,
			Value:     resValue.MilliValue(),
		}
	}
	return res
}
