package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ryanmt/cluster-resource-autoscaler/check"
	"github.com/ryanmt/cluster-resource-autoscaler/logging"
	"github.com/ryanmt/cluster-resource-autoscaler/utilization"
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

	logger.V(1).Info("Running...", "namespace", defaultNamespace)

	for {
		// For each configured check... calculate our current utlization against the goal
		config, err := check.FromFile("./test_config.json")
		if err != nil {
			logger.Error(err, "failure getting configuration from file")
			panic(err.Error())
		}

		for _, checkSpec := range config {
			rName := checkSpec.Resource().String()
			logger.V(1).Info("Received checkSpec", "resourceType", rName, "target", checkSpec.TargetUtilization)

			percentage, err := utilization.PercentageByResource(checkSpec.Resource())
			if err != nil {
				logger.Error(err, "failure getting cluster utilization for checkSpec")
				panic(err.Error())
			}

			usagePct := fmt.Sprintf("%.2f", percentage*100.0)
			targetPct := fmt.Sprintf("%.2f", checkSpec.TargetUtilization*100.0)
			logger.Info("Percent utilization", "resource", rName, "usage_pct", usagePct, "target_pct", targetPct)

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

		time.Sleep(10 * time.Second)
	}
}
