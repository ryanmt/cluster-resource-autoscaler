package check

import v1 "k8s.io/api/core/v1"

func SupportedResources() []v1.ResourceName {
	return []v1.ResourceName{
		v1.ResourceCPU,
		v1.ResourceMemory,
	}
}
