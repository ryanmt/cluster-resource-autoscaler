package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/ryanmt/cluster-resource-autoscaler/check"
	"github.com/ryanmt/cluster-resource-autoscaler/logging"
	"github.com/ryanmt/cluster-resource-autoscaler/scaler"
	"github.com/ryanmt/cluster-resource-autoscaler/utilization"
	"k8s.io/apimachinery/pkg/api/errors"
)

func main() {

	_, isDev := os.LookupEnv("RESOURCE_AUTOSCALER_TESTING_MODE")
	defaultNamespace := "resource-autoscaler"

	if !isDev {
		ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			defaultNamespace = "<indeterminate>"
		} else {
			defaultNamespace = string(ns)
		}
	}

	// Initialize clients and logging
	logger := logging.New(isDev)
	ctx := logging.NewContext(context.Background(), logger)
	utilization.Init(ctx, isDev)
	check.Init(ctx)
	scaler.Init(ctx, isDev)

	logger.V(1).Info("Running...", "namespace", defaultNamespace)

	for {
		var config []check.Spec
		var err error

		// TODO: Create this as configurable
		if isDev {
			config, err = check.FromFile("./test_config.json")
		} else {
			config, err = check.FromFile("./config/config.json")
		}

		// For each configured check... calculate our current utlization against the goal
		if err != nil {
			logger.Error(err, "failure getting configuration from file")
			panic(err.Error())
		}

		for _, checkSpec := range config {
			checkLogger := logger.WithValues("target", checkSpec.TargetKey(), "checkName", checkSpec.Name)
			// rName := checkSpec.Resource().String()
			// logger.V(1).Info("Received checkSpec", "resourceType", rName, "target", checkSpec.TargetUtilization)
			checkLogger.V(1).Info("checkSpec received")

			for _, rName := range check.SupportedResources() {
				scaleFactor := checkSpec.ResourceScaler(rName)
				var recommendedReplicas float64
				if scaleFactor != 0 {
					scalerLogger := checkLogger.WithValues("resource", rName)
					availableResource := utilization.CapacityByResource(rName)
					percentage := utilization.PercentageByResource(rName)

					usagePct := fmt.Sprintf("%.2f", percentage*100.0)
					targetPct := fmt.Sprintf("%.2f", checkSpec.ResourceScaler(rName))
					scalerLogger.V(1).Info("Percent utilization", "usage_pct", usagePct, "target_pct", targetPct)

					recommendedReplicas = math.Ceil(float64(availableResource) / checkSpec.ResourceScaler(rName))
					scalerLogger.V(1).Info("Scaling quotient", "available", availableResource, "scaler", checkSpec.ResourceScaler(rName), "calculatedReplicas", recommendedReplicas)
				} else {
					checkLogger.Info("Scaler does not apply", "resource", rName)
				}

				checkLogger.Info("Target", "TD", checkSpec.Target)
				currentReplicas, err := scaler.GetReplicas(checkSpec.Target)
				if errors.IsAlreadyExists(err) {
					checkLogger.Error(err, "Target doesn't exist", "target", checkSpec.TargetKey())
				} else if errors.IsNotFound(err) {
					checkLogger.Error(err, "Target doesn't exist", "target", checkSpec.TargetKey())
				} else {
					panic(err.Error())
				}
				checkLogger.V(1).Info("Current scale", "replica_count", currentReplicas)

				if currentReplicas != int32(recommendedReplicas) {
					// Recommend we do the upgrade
					checkLogger.Info("Recommended scaling", "action", fmt.Sprintf("%d=>%d", currentReplicas, int32(recommendedReplicas)))
				}
			}

			// Run each resource check
		}

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

		if isDev {
			// Running locally... don't sleep
			logger.V(1).Info("Development mode, exiting....")
			break
		}

		time.Sleep(30 * time.Second)
	}
}
