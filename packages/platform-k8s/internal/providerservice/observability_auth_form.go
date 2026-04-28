package providerservice

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"code-code.internal/go-contract/domainerror"
	observabilityv1 "code-code.internal/go-contract/observability/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
)

func (s *Server) observabilityAuthenticationCommand(
	ctx context.Context,
	providerID string,
	material *providerservicev1.SessionMaterial,
) (providers.UpdateObservabilityAuthenticationCommand, error) {
	provider, err := s.providers.Get(ctx, providerID)
	if err != nil {
		return providers.UpdateObservabilityAuthenticationCommand{}, err
	}
	form, err := s.observabilityInputForm(ctx, provider, material.GetSchemaId())
	if err != nil {
		return providers.UpdateObservabilityAuthenticationCommand{}, err
	}
	values, requiredKeys, err := normalizeObservabilityInputValues(form, material.GetValues())
	if err != nil {
		return providers.UpdateObservabilityAuthenticationCommand{}, err
	}
	return providers.UpdateObservabilityAuthenticationCommand{
		SchemaID:     strings.TrimSpace(form.GetSchemaId()),
		RequiredKeys: requiredKeys,
		Values:       values,
	}, nil
}

func (s *Server) observabilityInputForm(
	ctx context.Context,
	provider *managementv1.ProviderView,
	schemaID string,
) (*observabilityv1.ActiveQueryInputForm, error) {
	if provider == nil {
		return nil, domainerror.NewValidation("platformk8s/providerservice: provider is nil")
	}
	if s.vendorSupport == nil {
		return nil, domainerror.NewValidation("platformk8s/providerservice: vendor support is unavailable")
	}
	vendorID := providerVendorID(provider)
	if vendorID == "" {
		return nil, domainerror.NewValidation("platformk8s/providerservice: provider %q vendor_id is empty", provider.GetProviderId())
	}
	vendor, err := s.vendorSupport.Get(ctx, vendorID)
	if err != nil {
		return nil, err
	}
	form := findObservabilityInputForm(vendor, providerSurfaceIDs(provider), schemaID)
	if form == nil {
		if strings.TrimSpace(schemaID) != "" {
			return nil, domainerror.NewValidation("platformk8s/providerservice: observability input schema %q is not supported by provider %q", schemaID, provider.GetProviderId())
		}
		return nil, domainerror.NewValidation("platformk8s/providerservice: provider %q does not declare an observability input form", provider.GetProviderId())
	}
	return form, nil
}

func providerVendorID(provider *managementv1.ProviderView) string {
	if provider == nil {
		return ""
	}
	if vendorID := strings.TrimSpace(provider.GetVendorId()); vendorID != "" {
		return vendorID
	}
	for _, surface := range provider.GetSurfaces() {
		if vendorID := strings.TrimSpace(surface.GetVendorId()); vendorID != "" {
			return vendorID
		}
	}
	return ""
}

func providerSurfaceIDs(provider *managementv1.ProviderView) map[string]struct{} {
	out := map[string]struct{}{}
	for _, surface := range provider.GetSurfaces() {
		if surfaceID := strings.TrimSpace(surface.GetSurfaceId()); surfaceID != "" {
			out[surfaceID] = struct{}{}
		}
	}
	return out
}

func findObservabilityInputForm(
	vendor *supportv1.Vendor,
	surfaceIDs map[string]struct{},
	schemaID string,
) *observabilityv1.ActiveQueryInputForm {
	schemaID = strings.TrimSpace(schemaID)
	for _, binding := range vendor.GetProviderBindings() {
		bindingSurfaceID := strings.TrimSpace(binding.GetProviderBinding().GetSurfaceId())
		if len(surfaceIDs) > 0 {
			if _, ok := surfaceIDs[bindingSurfaceID]; !ok {
				continue
			}
		}
		for _, profile := range binding.GetObservability().GetProfiles() {
			form := profile.GetActiveQuery().GetInputForm()
			if form == nil || strings.TrimSpace(form.GetSchemaId()) == "" {
				continue
			}
			if schemaID == "" || strings.TrimSpace(form.GetSchemaId()) == schemaID {
				return form
			}
		}
	}
	return nil
}

func normalizeObservabilityInputValues(
	form *observabilityv1.ActiveQueryInputForm,
	values map[string]string,
) (map[string]string, []string, error) {
	if form == nil {
		return nil, nil, domainerror.NewValidation("platformk8s/providerservice: observability input form is nil")
	}
	fields := map[string]*observabilityv1.ActiveQueryInputField{}
	for _, field := range form.GetFields() {
		if fieldID := strings.TrimSpace(field.GetFieldId()); fieldID != "" {
			fields[fieldID] = field
		}
	}
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(value) == "" {
			continue
		}
		if _, ok := fields[key]; !ok {
			return nil, nil, domainerror.NewValidation("platformk8s/providerservice: observability input field %q is not declared by schema %q", key, form.GetSchemaId())
		}
	}

	out := map[string]string{}
	for _, field := range form.GetFields() {
		fieldID := strings.TrimSpace(field.GetFieldId())
		value := strings.TrimSpace(values[fieldID])
		if value == "" {
			continue
		}
		switch field.GetPersistence() {
		case observabilityv1.ActiveQueryInputPersistence_ACTIVE_QUERY_INPUT_PERSISTENCE_STORED_MATERIAL:
			out[fieldID] = value
		case observabilityv1.ActiveQueryInputPersistence_ACTIVE_QUERY_INPUT_PERSISTENCE_TRANSIENT:
			if err := applyTransientObservabilityInput(out, values, field, value); err != nil {
				return nil, nil, err
			}
		}
	}
	if len(out) == 0 {
		return nil, nil, domainerror.NewValidation("platformk8s/providerservice: observability authentication values are required")
	}
	return out, observabilityInputRequiredKeys(form), nil
}

func applyTransientObservabilityInput(
	out map[string]string,
	values map[string]string,
	field *observabilityv1.ActiveQueryInputField,
	value string,
) error {
	targetFieldID := strings.TrimSpace(field.GetTargetFieldId())
	switch field.GetTransform() {
	case observabilityv1.ActiveQueryInputValueTransform_ACTIVE_QUERY_INPUT_VALUE_TRANSFORM_MERGE_SET_COOKIE:
		base := strings.TrimSpace(out[targetFieldID])
		if base == "" {
			base = strings.TrimSpace(values[targetFieldID])
		}
		if base == "" {
			return domainerror.NewValidation(
				"platformk8s/providerservice: observability input field %q requires target field %q in the same submission",
				field.GetFieldId(),
				targetFieldID,
			)
		}
		if merged := mergeCookieHeader(base, value); merged != "" {
			out[targetFieldID] = merged
		}
		return nil
	default:
		return domainerror.NewValidation(
			"platformk8s/providerservice: unsupported observability input transform %s for field %q",
			field.GetTransform().String(),
			field.GetFieldId(),
		)
	}
}

func observabilityInputRequiredKeys(form *observabilityv1.ActiveQueryInputForm) []string {
	required := make([]string, 0, len(form.GetFields()))
	for _, field := range form.GetFields() {
		if field.GetPersistence() != observabilityv1.ActiveQueryInputPersistence_ACTIVE_QUERY_INPUT_PERSISTENCE_STORED_MATERIAL || !field.GetRequired() {
			continue
		}
		if fieldID := strings.TrimSpace(field.GetFieldId()); fieldID != "" {
			required = append(required, fieldID)
		}
	}
	return required
}

func mergeCookieHeader(requestCookie string, responseSetCookie string) string {
	cookies := map[string]string{}
	for _, pair := range strings.Split(requestCookie, ";") {
		applyCookiePair(cookies, pair)
	}
	for _, line := range strings.Split(responseSetCookie, "\n") {
		headerValue := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(headerValue), "set-cookie:") {
			headerValue = strings.TrimSpace(headerValue[len("set-cookie:"):])
		}
		if index := strings.Index(headerValue, ";"); index >= 0 {
			headerValue = headerValue[:index]
		}
		applyCookiePair(cookies, headerValue)
	}
	keys := make([]string, 0, len(cookies))
	for key := range cookies {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, cookies[key]))
	}
	return strings.Join(parts, "; ")
}

func applyCookiePair(cookies map[string]string, pair string) {
	key, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
	if !ok {
		return
	}
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" {
		return
	}
	if value == "" {
		delete(cookies, key)
		return
	}
	cookies[key] = value
}
