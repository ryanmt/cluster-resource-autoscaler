package health_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/ryanmt/cluster-resource-autoscaler/health"
)

func TestGCMaxPauseCheck(t *testing.T) {
	runtime.GC()
	if err := health.GCMaxPauseCheck(1 * time.Second)(); err != nil {
		t.Errorf("Given a large budget, the GC check shouldn't fail: '%v'", err)
	}

	if err := health.GCMaxPauseCheck(0)(); err == nil {
		t.Errorf("Given a budget of zero, the GC check should err: '%v'", err)
	}
}
