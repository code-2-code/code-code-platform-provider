package providercatalogs

import (
	"context"
	"fmt"
	"strings"

	supportv1 "code-code.internal/go-contract/platform/support/v1"
	vendorsupport "code-code.internal/platform-k8s/internal/supportservice/vendors/support"
	"google.golang.org/protobuf/proto"
)

// VendorSupportReader lists vendor support definitions.
type VendorSupportReader interface {
	List(context.Context) ([]*supportv1.Vendor, error)
}

func registerVendorSources(ctx context.Context, registry *catalogSourceRegistry, support VendorSupportReader) error {
	if support == nil {
		return fmt.Errorf("platformk8s/providercatalogs: vendor support reader is nil")
	}
	vendors, err := support.List(ctx)
	if err != nil {
		return err
	}
	for _, vendor := range vendors {
		if !hasCatalogCapability(vendor) {
			continue
		}
		vendorID := strings.TrimSpace(vendor.GetVendor().GetVendorId())
		if vendorID == "" {
			return fmt.Errorf("platformk8s/providercatalogs: vendor support id is empty")
		}
		if err := registry.register(&vendorCatalogSource{
			sourceRef: newCatalogSourceRef("vendor." + vendorID),
			vendor:    proto.Clone(vendor).(*supportv1.Vendor),
		}); err != nil {
			return err
		}
	}
	return nil
}

func hasCatalogCapability(vendor *supportv1.Vendor) bool {
	if vendor == nil {
		return false
	}
	for _, binding := range vendor.GetProviderBindings() {
		if vendorsupport.SupportsModelCatalogProbe(binding) {
			return true
		}
	}
	return false
}

type vendorCatalogSource struct {
	sourceRef catalogSourceRef
	vendor    *supportv1.Vendor
}

func (s *vendorCatalogSource) ref() catalogSourceRef {
	return s.sourceRef
}
