package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"

	"github.com/heptiolabs/healthcheck"
	"github.com/ryanmt/cluster-resource-autoscaler/check"
	"github.com/ryanmt/cluster-resource-autoscaler/health"
	"github.com/ryanmt/cluster-resource-autoscaler/kubeapi"
	"github.com/ryanmt/cluster-resource-autoscaler/logging"
	"github.com/ryanmt/cluster-resource-autoscaler/scaler"
	"github.com/ryanmt/cluster-resource-autoscaler/utilization"
	"k8s.io/apimachinery/pkg/api/errors"
)

const healthCheckPort = ":8085"

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

	// Initialize generic api clients
	kubeapi.Init(ctx, isDev)

	// Init packages to make them logging empowered or create clients if needed
	utilization.Init(ctx)
	check.Init(ctx)
	scaler.Init(ctx)

	logger.V(2).Info("Running...", "namespace", defaultNamespace)

	if isDev {
		logger.Info("Not running health check... Dev mode")
	} else {
		logger.Info("Dropping into the health check code now")
		healthHandler := healthcheck.NewHandler()

		// Check that we aren't leaking goroutines
		healthHandler.AddLivenessCheck("goroutine-threshold", healthcheck.GoroutineCountCheck(1000))

		// TODO: Check we are able to talk to the cluster
		clusterURL, _ := url.Parse(kubeapi.Config.Host)
		pathURL, _ := url.Parse("/readyz")
		checkURL := clusterURL.ResolveReference(pathURL)
		logger.Info("Cluster testing URL", "url", checkURL.String())
		// healthHandler.AddReadinessCheck("cluster-connectivity", healthcheck.HTTPGetCheck(checkURL.String(), 11*time.Second))
		healthHandler.AddReadinessCheck("GC-timing", health.GCMaxPauseCheck(1*time.Second))

		go func() {
			// err := http.ListenAndServe(healthCheckPort, logging.LoggingMiddleware(healthHandler, logger))
			err := http.ListenAndServe(healthCheckPort, healthHandler)
			if err != nil {
				logger.Error(err, "ListenAndServe failure for health check handler")
				panic(err)
			}
		}()

		go func() {
			for {
				// Debugging... capture some example requests
				req, err := http.NewRequest("GET", "//127.0.0.1:8085/ready", nil)
				if err != nil {
					logger.Error(err, "There was an error in checking the first request!!")
					panic(err)
				}
				rr := httptest.NewRecorder()
				healthHandler.ServeHTTP(rr, req)

				resp := rr.Result()
				body, _ := io.ReadAll(resp.Body)
				logger.V(2).Info("status", "code", resp.StatusCode, "content-type", resp.Header.Get("Content-Type"), "body", string(body))
				// Check the status code is what we expect.
				if status := rr.Code; status != http.StatusOK {
					logger.Error(err, fmt.Sprintf("handler returned wrong status code: got %v want %v", status, http.StatusOK))
				}

				time.Sleep(60 * time.Second)
			}
		}()
	}

	for {
		var config []check.Spec
		var err error

		// TODO: Make this configurable
		if isDev {
			config, err = check.FromFile("./test_config.json")
		} else {
			config, err = check.FromFile("./config/config.json")
		}

		if err != nil {
			logger.Error(err, "failure getting configuration from file")
			panic(err.Error())
		}
		// For each configured check... calculate our current utilization against the goal
		// We can iterate on the check, because only one can apply to a given
		// target.  Any duplicate targets are ignored at the configuration layer.
		for _, checkSpec := range config {
			checkLogger := logger.WithValues("target", checkSpec.TargetKey(), "checkName", checkSpec.Name)
			checkLogger.V(2).Info("checkSpec received")

			var recommendations []float64

			for _, rName := range check.SupportedResources() {
				scaleFactor := checkSpec.ResourceScaler(rName)
				checkLogger.V(1).Info("scaleFactor calculation", "scaleFactor", scaleFactor)
				if scaleFactor != 0 {
					scalerLogger := checkLogger.WithValues("resource", rName)
					availableResource := utilization.CapacityByResource(rName)
					percentage := utilization.PercentageByResource(rName)

					usagePct := fmt.Sprintf("%.2f", percentage*100.0)
					targetPct := fmt.Sprintf("%.2f", checkSpec.ResourceScaler(rName))
					scalerLogger.V(2).Info("Percent utilization", "usage_pct", usagePct, "target_pct", targetPct)

					newRecommendation := math.Ceil(float64(availableResource) / checkSpec.ResourceScaler(rName))
					scalerLogger.V(2).Info("Scaling quotient", "available", availableResource, "scaler", checkSpec.ResourceScaler(rName), "calculatedReplicas", newRecommendation)

					recommendations = append(recommendations, newRecommendation)
				} else {
					checkLogger.V(1).Info("Scaler does not apply", "resource", rName)
				}
			}

			// New recommendedReplicas is the highest of all recommendations
			// TODO: Make this behavior configurable, i.e. "max", "min", "geometric_mean"
			var recommendedReplicas float64
			for _, v := range recommendations {
				recommendedReplicas = math.Max(recommendedReplicas, v)
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

			checkLogger.Info("Current scale", "replica_count", currentReplicas)

			if currentReplicas != int32(recommendedReplicas) {
				// Recommend we do the upgrade, and if not DRYRUN, do it
				checkLogger.Info("Recommended scaling (based on all inputs)", "action", fmt.Sprintf("%d=>%d", currentReplicas, int32(recommendedReplicas)))

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
		if isDev {
			// Running locally... don't sleep
			logger.V(2).Info("Development mode, exiting....")
			break
		}

		time.Sleep(30 * time.Second)
	}
}
