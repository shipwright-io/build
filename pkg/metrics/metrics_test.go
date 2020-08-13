package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redhat-developer/build/pkg/config"
)

func TestBuildRunMetrics(t *testing.T) {
	testCases := []struct {
		name      string
		namespace string
		strategy  string
	}{
		{
			name:      "buildpacks",
			namespace: "test",
			strategy:  "buildpacks",
		},
		{
			name:      "kaniko",
			namespace: "default",
			strategy:  "kaniko",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			InitPrometheus(config.NewDefaultConfig())

			BuildCountInc(tc.strategy)
			BuildRunCountInc(tc.strategy)
			buildRunEstablishTime := time.Duration(1) * time.Second
			buildRunExecutionTime := time.Duration(100) * time.Second
			BuildRunEstablishObserve(tc.strategy, tc.namespace, buildRunEstablishTime)
			BuildRunCompletionObserve(tc.strategy, tc.namespace, buildRunExecutionTime)

			buildCount, err := buildCount.GetMetricWithLabelValues(tc.strategy)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if testutil.ToFloat64(buildCount) != float64(1) {
				t.Errorf("expected build count to equal %f, got %f", float64(1), testutil.ToFloat64(buildCount))
			}

			buildRunCount, err := buildRunCount.GetMetricWithLabelValues(tc.strategy)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if testutil.ToFloat64(buildRunCount) != float64(1) {
				t.Errorf("expected buildRun count to equal %f, got %f", float64(1), testutil.ToFloat64(buildRunCount))
			}

			buildRunEstablishDuration, err := buildRunEstablishDuration.GetMetricWithLabelValues(tc.strategy, tc.namespace)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buildRunEstablishDuration == nil {
				t.Error("expected buildRunEstablishDuration to not be nil")
			}

			buildRunCompletionDuration, err := buildRunCompletionDuration.GetMetricWithLabelValues(tc.strategy, tc.namespace)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buildRunCompletionDuration == nil {
				t.Error("expected buildRunCompletionDuration to not be nil")
			}
		})
	}
}
