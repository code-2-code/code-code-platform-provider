package providerconnect

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"code-code.internal/go-contract/domainerror"
)

type providerConnectQueries struct {
	providers providerReader
	metadata  surfaceMetadataReader
}

func newProviderConnectQueries(
	providers providerReader,
	metadata surfaceMetadataReader,
) *providerConnectQueries {
	return &providerConnectQueries{
		providers: providers,
		metadata:  metadata,
	}
}

func (q *providerConnectQueries) FindProviderBySurface(ctx context.Context, surfaceID string) (*ProviderView, error) {
	if q == nil || q.providers == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider query is not configured")
	}
	items, err := q.providers.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.GetSurfaceId() == surfaceID {
			return item, nil
		}
	}
	return nil, domainerror.NewNotFound("platformk8s/providerconnect: provider with surface %q not found", surfaceID)
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
