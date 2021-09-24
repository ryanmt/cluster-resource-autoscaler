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

	logger.V(2).Info("Running...", "namespace", defaultNamespace)

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
			checkLogger.V(2).Info("checkSpec received")

			for _, rName := range check.SupportedResources() {
				scaleFactor := checkSpec.ResourceScaler(rName)
				var recommendedReplicas float64
				if scaleFactor != 0 {
					scalerLogger := checkLogger.WithValues("resource", rName)
					availableResource := utilization.CapacityByResource(rName)
					percentage := utilization.PercentageByResource(rName)

					usagePct := fmt.Sprintf("%.2f", percentage*100.0)
					targetPct := fmt.Sprintf("%.2f", checkSpec.ResourceScaler(rName))
					scalerLogger.V(2).Info("Percent utilization", "usage_pct", usagePct, "target_pct", targetPct)

					recommendedReplicas = math.Ceil(float64(availableResource) / checkSpec.ResourceScaler(rName))
					scalerLogger.V(2).Info("Scaling quotient", "available", availableResource, "scaler", checkSpec.ResourceScaler(rName), "calculatedReplicas", recommendedReplicas)
				} else {
					checkLogger.V(1).Info("Scaler does not apply", "resource", rName)
				}

				currentReplicas, err := scaler.GetReplicas(checkSpec.Target)
				if err != nil {
					if errors.IsAlreadyExists(err) {
						checkLogger.Error(err, "Target already exists", "target", checkSpec.TargetKey())
					} else if errors.IsNotFound(err) {
						checkLogger.Error(err, "Target doesn't exist", "target", checkSpec.TargetKey())
					} else {
						panic(err.Error())
					}
					continue
				}

				checkLogger.V(2).Info("Current scale", "replica_count", currentReplicas)

				if currentReplicas != int32(recommendedReplicas) {
					// Recommend we do the upgrade, and if not DRYRUN, do it
					checkLogger.Info("Recommended scaling", "action", fmt.Sprintf("%d=>%d", currentReplicas, int32(recommendedReplicas)))

					if _, ok := os.LookupEnv("CRA_DRYRUN"); ok {
						continue
					}

					oldReplicas, err := scaler.UpdateReplicas(checkSpec.Target, int32(recommendedReplicas))
					if err != nil {
						checkLogger.Error(err, "Error in UpdateReplicas")
						continue
					}
					checkLogger.Info("Updated target", "oldReplicas", oldReplicas, "newReplicas", recommendedReplicas)
				}
			}
		}
		if isDev {
			// Running locally... don't sleep
			logger.V(2).Info("Development mode, exiting....")
			break
		}

		time.Sleep(30 * time.Second)
	}
}
