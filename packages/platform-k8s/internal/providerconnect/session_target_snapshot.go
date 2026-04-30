package providerconnect

import (
	"fmt"
	"strings"

	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type sessionTargetSnapshot struct {
	AddMethod          AddMethod `json:"addMethod"`
	DisplayName        string    `json:"displayName"`
	VendorID           string    `json:"vendorId"`
	CLIID              string    `json:"cliId"`
	ProviderSurfaceID  string    `json:"providerSurfaceId"`
	TargetCredentialID string    `json:"targetCredentialId"`
	TargetProviderID   string    `json:"targetProviderId"`
	Runtime            string    `json:"runtime"`
}

func newSessionTargetSnapshot(target *connectTarget) (sessionTargetSnapshot, error) {
	if target == nil {
		return sessionTargetSnapshot{}, fmt.Errorf("platformk8s/providerconnect: session target is nil")
	}
	runtimeJSON, err := encodeProviderSurfaceRuntime(target.RuntimeTemplate)
	if err != nil {
		return sessionTargetSnapshot{}, err
	}
	return sessionTargetSnapshot{
		AddMethod:          target.AddMethod,
		DisplayName:        strings.TrimSpace(target.DisplayName),
		VendorID:           strings.TrimSpace(target.VendorID),
		CLIID:              strings.TrimSpace(target.CLIID),
		ProviderSurfaceID:  strings.TrimSpace(target.SurfaceID),
		TargetCredentialID: strings.TrimSpace(target.TargetCredentialID),
		TargetProviderID:   strings.TrimSpace(target.TargetProviderID),
		Runtime:            runtimeJSON,
	}, nil
}

func (s sessionTargetSnapshot) needsFinalize(connectedSurfaceID string) bool {
	return strings.TrimSpace(s.ProviderSurfaceID) != "" && strings.TrimSpace(connectedSurfaceID) == ""
}

func (s sessionTargetSnapshot) runtime() (*providerv1.ProviderSurfaceRuntime, error) {
	return decodeProviderSurfaceRuntime(s.Runtime)
}

func (s sessionTargetSnapshot) target(runtime *providerv1.ProviderSurfaceRuntime) *connectTarget {
	return newConnectTargetWithIDs(
		s.AddMethod,
		s.DisplayName,
		s.VendorID,
		s.CLIID,
		s.ProviderSurfaceID,
		s.TargetCredentialID,
		s.TargetProviderID,
		runtime,
	)
}

func encodeProviderSurfaceRuntime(runtime *providerv1.ProviderSurfaceRuntime) (string, error) {
	if runtime == nil {
		return "", nil
	}
	runtimeJSON, err := protojson.MarshalOptions{UseProtoNames: false}.Marshal(runtime)
	if err != nil {
		return "", fmt.Errorf("platformk8s/providerconnect: marshal provider surface runtime: %w", err)
	}
	return string(runtimeJSON), nil
}

func decodeProviderSurfaceRuntime(raw string) (*providerv1.ProviderSurfaceRuntime, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	runtime := &providerv1.ProviderSurfaceRuntime{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal([]byte(raw), runtime); err != nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: decode provider surface runtime: %w", err)
	}
	return runtime, nil
}
