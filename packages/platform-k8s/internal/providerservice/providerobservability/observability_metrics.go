package providerobservability

import (
	"fmt"
	"regexp"
	"sync"

	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/metric"
)

// observabilityMetrics owns provider-collected quota gauge instruments.
type observabilityMetrics struct {
	ownerLabel string
	meter      otelmetric.Meter

	collectedMu     sync.Mutex
	collectedGauges map[string]collectedGauge
}

type collectedGauge struct {
	gauge otelmetric.Float64Gauge
}

var (
	observabilityMetricNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.\-/]{0,254}$`)
	observabilityLabelNamePattern  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// newObservabilityMetrics creates a quota gauge recorder. Probe run status is
// emitted by provider-orchestration, which owns probe preflight and persistence.
func newObservabilityMetrics(metricPrefix string, ownerLabel string) (*observabilityMetrics, error) {
	meter := otel.Meter("platform-k8s/providerobservability")
	if metricPrefix == "" {
		return nil, fmt.Errorf("providerobservability: metric prefix is empty")
	}
	return &observabilityMetrics{
		ownerLabel:      ownerLabel,
		meter:           meter,
		collectedGauges: map[string]collectedGauge{},
	}, nil
}

func newCredentialsGauge(meter otelmetric.Meter, name string, description string) (otelmetric.Float64Gauge, error) {
	gauge, err := meter.Float64Gauge(name, otelmetric.WithDescription(description))
	if err != nil {
		return nil, fmt.Errorf("providerobservability: create gauge %q: %w", name, err)
	}
	return gauge, nil
}
