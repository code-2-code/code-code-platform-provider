package providerservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providerobservability"
)

const (
	providerCredentialReadyWaitTimeout  = 90 * time.Second
	providerCredentialReadyPollInterval = 2 * time.Second
)

func (s *Server) runProviderCatalogDiscovery(ctx context.Context, providerIDs []string) error {
	if s == nil || s.catalogDiscovery == nil {
		return fmt.Errorf("platformk8s/providerservice: provider catalog discovery is not initialized")
	}
	if ids := normalizedProviderIDs(providerIDs); len(ids) > 0 {
		if err := s.waitProviderCredentialsReady(ctx, ids); err != nil {
			return err
		}
	}
	return s.catalogDiscovery.Sync(ctx, providerIDs)
}

func (s *Server) DiscoverProviderCatalogs(ctx context.Context, providerIDs []string) error {
	return s.runProviderCatalogDiscovery(ctx, normalizedProviderIDs(providerIDs))
}

// runProviderCatalogBinding is a no-op: model ref binding is now done inline
// during catalog materialization via modelidentity resolution.
func (s *Server) runProviderCatalogBinding(_ context.Context) error {
	return nil
}

func (s *Server) runProviderObservabilityProbe(ctx context.Context, providerIDs []string, trigger providerobservability.Trigger) error {
	ids := append([]string(nil), providerIDs...)
	if s == nil || s.providerObservability == nil {
		return fmt.Errorf("platformk8s/providerservice: provider observability is not initialized")
	}
	if err := s.waitProviderCredentialsReady(ctx, ids); err != nil {
		return err
	}
	var errs []error
	for _, providerID := range ids {
		_, err := s.providerObservability.ProbeProvider(ctx, providerID, trigger)
		if err != nil {
			errs = append(errs, fmt.Errorf("probe provider %q: %w", providerID, err))
		}
		if ctx.Err() != nil {
			errs = append(errs, ctx.Err())
			break
		}
	}
	return errors.Join(errs...)
}

func (s *Server) waitProviderCredentialsReady(ctx context.Context, providerIDs []string) error {
	if len(providerIDs) == 0 {
		return nil
	}
	notReady, err := s.notReadyProviderCredentials(ctx, providerIDs)
	if err != nil || len(notReady) == 0 {
		return err
	}
	deadline := time.NewTimer(providerCredentialReadyWaitTimeout)
	ticker := time.NewTicker(providerCredentialReadyPollInterval)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("platformk8s/providerservice: provider credentials not ready: %s", strings.Join(notReady, ", "))
		case <-ticker.C:
			notReady, err = s.notReadyProviderCredentials(ctx, providerIDs)
			if err != nil || len(notReady) == 0 {
				return err
			}
		}
	}
}

func (s *Server) notReadyProviderCredentials(ctx context.Context, providerIDs []string) ([]string, error) {
	if s == nil || s.authClient == nil {
		return nil, fmt.Errorf("platformk8s/providerservice: auth service client is not initialized")
	}
	credentials, err := s.authClient.ListCredentials(ctx, &authv1.ListCredentialsRequest{})
	if err != nil {
		return nil, err
	}
	ready := credentialReadiness(credentials.GetItems())
	notReady := []string{}
	for _, providerID := range providerIDs {
		provider, err := s.providers.Get(ctx, providerID)
		if err != nil {
			return nil, err
		}
		credentialID := providerCredentialID(provider)
		if credentialID == "" || !ready[credentialID] {
			notReady = append(notReady, providerID)
		}
	}
	return notReady, nil
}

func credentialReadiness(items []*managementv1.CredentialView) map[string]bool {
	out := map[string]bool{}
	for _, item := range items {
		credentialID := strings.TrimSpace(item.GetCredentialId())
		if credentialID == "" {
			continue
		}
		out[credentialID] = item.GetStatus().GetMaterialReady()
	}
	return out
}

func providerCredentialID(provider *managementv1.ProviderView) string {
	if provider == nil {
		return ""
	}
	if credentialID := strings.TrimSpace(provider.GetProviderCredentialId()); credentialID != "" {
		return credentialID
	}
	for _, surface := range provider.GetSurfaces() {
		if credentialID := strings.TrimSpace(surface.GetProviderCredentialId()); credentialID != "" {
			return credentialID
		}
	}
	return ""
}
