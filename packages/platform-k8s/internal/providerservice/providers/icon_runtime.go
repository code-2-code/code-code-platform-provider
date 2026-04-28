package providers

import (
	"context"
	"strings"
)

type providerIconRuntime struct {
	vendors vendorReferenceService
	cliDefs cliDefinitionReferenceService
}

func newProviderIconRuntime(
	vendors vendorReferenceService,
	cliDefs cliDefinitionReferenceService,
) providerIconRuntime {
	return providerIconRuntime{
		vendors: vendors,
		cliDefs: cliDefs,
	}
}

func (r providerIconRuntime) Apply(ctx context.Context, projections []*ProviderProjection) []*ProviderProjection {
	if len(projections) == 0 {
		return nil
	}
	vendorIcons := r.vendorIconsByID(ctx)
	cliIcons := r.cliIconsByID(ctx)
	items := make([]*ProviderProjection, 0, len(projections))
	for _, projection := range projections {
		items = append(items, projection.WithIconURL(projection.IconURL(vendorIcons, cliIcons)))
	}
	return items
}

func (r providerIconRuntime) vendorIconsByID(ctx context.Context) map[string]string {
	if r.vendors == nil {
		return nil
	}
	items, err := r.vendors.List(ctx)
	if err != nil {
		return nil
	}
	out := make(map[string]string, len(items))
	for _, item := range items {
		vendorID := strings.TrimSpace(item.GetVendorId())
		if vendorID == "" {
			continue
		}
		out[vendorID] = strings.TrimSpace(item.GetIconUrl())
	}
	return out
}

func (r providerIconRuntime) cliIconsByID(ctx context.Context) map[string]string {
	if r.cliDefs == nil {
		return nil
	}
	items, err := r.cliDefs.List(ctx)
	if err != nil {
		return nil
	}
	out := make(map[string]string, len(items))
	for _, item := range items {
		cliID := strings.TrimSpace(item.GetCliId())
		if cliID == "" {
			continue
		}
		out[cliID] = strings.TrimSpace(item.GetIconUrl())
	}
	return out
}
