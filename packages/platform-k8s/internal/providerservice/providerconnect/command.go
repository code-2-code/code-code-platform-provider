package providerconnect

import (
	"strings"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	"code-code.internal/go-contract/domainerror"
)

// ConnectCommand carries one normalized provider connect request.
type ConnectCommand struct {
	addMethod   AddMethod
	displayName string
	vendorID    string
	cliID       string
	apiKey      *APIKeyConnectInput
}

// NewConnectCommand validates and clones one provider connect command input.
func NewConnectCommand(input ConnectCommandInput) (*ConnectCommand, error) {
	command := &ConnectCommand{
		addMethod:   input.AddMethod,
		displayName: strings.TrimSpace(input.DisplayName),
		vendorID:    strings.TrimSpace(input.VendorID),
		cliID:       strings.TrimSpace(input.CLIID),
		apiKey:      cloneAPIKeyConnectInput(input.APIKey),
	}
	if command.AddMethod() == AddMethodUnspecified {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: add_method is required")
	}
	return command, nil
}

func (c *ConnectCommand) AddMethod() AddMethod {
	if c == nil {
		return AddMethodUnspecified
	}
	return c.addMethod
}

func (c *ConnectCommand) DisplayName() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.displayName)
}

func (c *ConnectCommand) DisplayNameOr(fallback string) string {
	if displayName := c.DisplayName(); displayName != "" {
		return displayName
	}
	return strings.TrimSpace(fallback)
}

func (c *ConnectCommand) VendorID() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.vendorID)
}

func (c *ConnectCommand) CLIID() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.cliID)
}

func (c *ConnectCommand) APIKeyInput() *APIKeyConnectInput {
	if c == nil || c.apiKey == nil {
		return nil
	}
	return cloneAPIKeyConnectInput(c.apiKey)
}

func (c *ConnectCommand) APIKeyValue() string {
	if input := c.APIKeyInput(); input != nil {
		return strings.TrimSpace(input.APIKey)
	}
	return ""
}

func (c *ConnectCommand) SurfaceModelCatalogs() []*ProviderSurfaceBindingModelCatalogInput {
	if input := c.APIKeyInput(); input != nil {
		return input.SurfaceModelCatalogs
	}
	return nil
}

func (c *ConnectCommand) IsVendorAPIKey() bool {
	return c.VendorID() != ""
}

func (c *ConnectCommand) ValidateAPIKey() error {
	material := c.APIKeyInput()
	if material == nil || strings.TrimSpace(material.APIKey) == "" {
		return domainerror.NewValidation("platformk8s/providerconnect: api key is required")
	}
	if c.IsVendorAPIKey() {
		return c.validateVendorAPIKey()
	}
	return c.validateCustomAPIKey()
}

func (c *ConnectCommand) ValidateCLIOAuth() error {
	if c.CLIID() == "" {
		return domainerror.NewValidation("platformk8s/providerconnect: cli_id is required for CLI OAuth")
	}
	return nil
}

func (c *ConnectCommand) validateVendorAPIKey() error {
	material := c.APIKeyInput()
	if strings.TrimSpace(material.BaseURL) != "" {
		return domainerror.NewValidation("platformk8s/providerconnect: vendor API key connect does not accept base_url")
	}
	if material.Protocol != apiprotocolv1.Protocol_PROTOCOL_UNSPECIFIED {
		return domainerror.NewValidation("platformk8s/providerconnect: vendor API key connect does not accept protocol")
	}
	return nil
}

func (c *ConnectCommand) validateCustomAPIKey() error {
	material := c.APIKeyInput()
	if strings.TrimSpace(material.BaseURL) == "" {
		return domainerror.NewValidation("platformk8s/providerconnect: base_url is required for custom API key connect")
	}
	if material.Protocol == apiprotocolv1.Protocol_PROTOCOL_UNSPECIFIED {
		return domainerror.NewValidation("platformk8s/providerconnect: protocol is required for custom API key connect")
	}
	return nil
}
