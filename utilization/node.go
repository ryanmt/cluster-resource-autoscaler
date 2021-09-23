package utilization

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/ryanmt/cluster-resource-autoscaler/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"

	"k8s.io/client-go/rest"
)

type MetricDatum struct {
	Timestamp time.Time
	Window    time.Duration
	Value     int64
}

type Metrics map[string]MetricDatum

var logger logr.Logger
var metricClient *resourceclient.MetricsV1beta1Client
var k8client *kubernetes.Clientset
var ctx context.Context

func Init(initCtx context.Context, isDev bool) {
	var err error
	var config *rest.Config

	logger = logging.FromContextOrDiscard(initCtx)
	ctx = initCtx

	if isDev {
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	metricClient = resourceclient.NewForConfigOrDie(config)
	k8client = kubernetes.NewForConfigOrDie(config)
}

func PercentageByResource(resource corev1.ResourceName) (float64, error) {
	nodeMetrics, err := metricClient.NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error getting Metrics: %v\n", err.Error())
	}

	nMetrics := getMetrics(nodeMetrics.Items, resource)

	var nodeResourceUsage, nodeResourceAvailable int64

	nodeResourceAvailable = getNodeResourcesAllocatable(resource)
	for k, m := range nMetrics {
		logger.V(2).Info("Found node metric", "resource", resource.String(), "node", k, "value", m.Value)
		nodeResourceUsage += m.Value
	}
	logger.V(1).Info("Node utilization in millicores", "value", nodeResourceUsage)
	logger.V(1).Info("Node resources allocated (millicores)", "value", nodeResourceAvailable)

	return float64(nodeResourceUsage) / float64(nodeResourceAvailable), nil
}

// resourceNames
type CoreResourceNames = []corev1.ResourceName

const CPU = corev1.ResourceCPU

func ResourceNames() []corev1.ResourceName {
	return []corev1.ResourceName{
		corev1.ResourceCPU,
		corev1.ResourceEphemeralStorage,
		corev1.ResourceHugePagesPrefix,
		corev1.ResourceMemory,
		corev1.ResourceStorage,
	}
}

func getNodeResourcesAllocatable(resourceName corev1.ResourceName) int64 {
	n, err := k8client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
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

func getMetrics(rawNodeMetrics []v1beta1.NodeMetrics, resource corev1.ResourceName) Metrics {
	res := make(Metrics, len(rawNodeMetrics))
	for _, m := range rawNodeMetrics {
		resValue, found := m.Usage[resource]
		if !found {
			logger.V(1).Info("Missing resource metric", "resourceName", resource.String(), "namespace", m.Namespace, "name", m.Name)
			break
		}
		res[m.Name] = MetricDatum{
			Timestamp: m.Timestamp.Time,
			Window:    m.Window.Duration,
			Value:     resValue.MilliValue(),
		}
	}
	return res
}
