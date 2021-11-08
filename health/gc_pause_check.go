package health

import (
	"fmt"
	"runtime"
	"time"

	"github.com/heptiolabs/healthcheck"
)

// derived from: https://github.com/heptiolabs/healthcheck/pull/11/commits/ee1efa91af44f422373eef34331336a5e399c32c
// GCMaxPauseCheck returns a Check that fails if any recent Go garbage
// collection pause exceeds the provided threshold.
func GCMaxPauseCheck(threshold time.Duration) healthcheck.Check {
	thresholdNanoseconds := uint64(threshold.Nanoseconds())
	return func() error {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		for _, pause := range stats.PauseNs {
			if pause > thresholdNanoseconds {
				return fmt.Errorf("recent GC cycle took %s > %s", time.Duration(pause), threshold)
			}
		}
		return nil
	}
}
