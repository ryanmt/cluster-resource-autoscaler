package check

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	corev1 "k8s.io/api/core/v1"
)

// Support checks from CM
// Support checks from JSON

type ScalingTarget struct {
	Name      string
	Namespace string
}

type Resource struct {
	Name                    string
	ReplicaScalingThreshold string // soemthing like `500m` as the "amount of cluster compute to base replicas on"
}

type Spec struct {
	ResourceName       string        // Target resource, ala CPU, Memory
	ResourcePerReplica string        // How much resource per replica of the deployment
	TargetDeployment   ScalingTarget // What deployment to scale
	TargetUtilization  float64       // Target utilization for the resourceName
}

func (s *Spec) Resource() corev1.ResourceName {
	return lookupResourceByName(s.ResourceName)
}

func FromFile(jsonFile string) ([]Spec, error) {
	file, err := os.Open(jsonFile)
	if err != nil {
		return nil, err
	}

	return FromReader(file)
}

func FromReader(r io.Reader) ([]Spec, error) {
	var specList []Spec
	var err error

	d := json.NewDecoder(r)

	// Read opening bracket
	_, err = d.Token()
	if err != nil {
		return specList, err
	}

	// TODO: Not sure if the the streaming version is a good idea in complexity...
	// just feels safer to model a bounded JSON implementation
	for d.More() {
		var s Spec
		err := d.Decode(&s)
		if err != nil {
			// Blah blah blah
			fmt.Printf("AHHHH!!! things didn't decode!!!")
			return specList, err
		}

		specList = append(specList, s)
	}
	_, err = d.Token() // Should just be the final bracket
	if err != nil {
		return specList, err
	}

	// TODO: validate each resourceName is in set
	for _, s := range specList {
		if !validResourceName(s.ResourceName) {
			fmt.Printf("Invalid resource name: %s\n", s.ResourceName)
		}
	}

	return specList, nil
}

var validResourceNames = []string{"cpu", "memory", "ephemeral_storage", "huge_pages"}

func validResourceName(name string) bool {
	for _, n := range validResourceNames {
		if n == name {
			return true
		}
	}

	return false
}

func lookupResourceByName(name string) corev1.ResourceName {
	switch name {
	case "cpu":
		return corev1.ResourceCPU
	case "memory":
		return corev1.ResourceMemory
	case "ephemeral_storage":
		return corev1.ResourceEphemeralStorage
	case "huge_pages":
		return corev1.ResourceHugePagesPrefix
	default:
		return corev1.ResourceCPU
	}
}
