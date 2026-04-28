package providerconnect

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"code-code.internal/go-contract/domainerror"
)

type providerConnectQueries struct {
	surfaces  providerSurfaceBindingReader
	providers providerReader
	metadata  surfaceMetadataReader
}

func newProviderConnectQueries(
	surfaces providerSurfaceBindingReader,
	providers providerReader,
	metadata surfaceMetadataReader,
) *providerConnectQueries {
	return &providerConnectQueries{
		surfaces:  surfaces,
		providers: providers,
		metadata:  metadata,
	}
}

func (q *providerConnectQueries) FindSurface(ctx context.Context, surfaceID string) (*ProviderSurfaceBindingView, error) {
	if q == nil || q.surfaces == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider surface binding query is not configured")
	}
	items, err := q.surfaces.ListProviderSurfaceBindings(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.GetSurfaceId() == surfaceID {
			return item, nil
		}
	}
	return nil, domainerror.NewNotFound("platformk8s/providerconnect: provider surface binding %q not found", surfaceID)
}

func (q *providerConnectQueries) FindProvider(ctx context.Context, providerID string) (*ProviderView, error) {
	if q == nil || q.providers == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider service is not configured")
	}
	return q.providers.Get(ctx, providerID)
}

func (q *providerConnectQueries) LoadSurfaceMetadata(
	ctx context.Context,
	surfaceID string,
) (*connectSurfaceMetadata, error) {
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider surface_id is required")
	}
	if q == nil || q.metadata == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider surface query is not configured")
	}
	surface, err := q.metadata.Get(ctx, surfaceID)
	if err != nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: get provider surface %q: %w", surfaceID, err)
	}
	return newConnectSurfaceMetadata(surface)
}

func isNotFound(err error) bool {
	var notFound *domainerror.NotFoundError
	return errors.As(err, &notFound)
}

func isAlreadyExists(err error) bool {
	var alreadyExists *domainerror.AlreadyExistsError
	return errors.As(err, &alreadyExists)
}
