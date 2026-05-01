package providerconnect

import (
	"strings"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	"code-code.internal/go-contract/domainerror"
	"code-code.internal/platform-k8s/internal/platform/providersurfaces/registry"
)

// ConnectCommand carries one normalized provider connect request.
type ConnectCommand struct {
	addMethod   AddMethod
	displayName string
	cliID       string
	surfaceID   string
	apiKey      *APIKeyConnectInput
}

// NewConnectCommand validates and clones one provider connect command input.
func NewConnectCommand(input ConnectCommandInput) (*ConnectCommand, error) {
	command := &ConnectCommand{
		addMethod:   input.AddMethod,
		displayName: strings.TrimSpace(input.DisplayName),
		cliID:       strings.TrimSpace(input.CLIID),
		surfaceID:   strings.TrimSpace(input.SurfaceID),
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

func (c *ConnectCommand) CLIID() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.cliID)
}

func (c *ConnectCommand) SurfaceID() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.surfaceID)
}

func (c *ConnectCommand) APIKeyInput() *APIKeyConnectInput {
	if c == nil || c.apiKey == nil {
		return nil
	}
	return cloneAPIKeyConnectInput(c.apiKey)
}

func (c *ConnectCommand) CredentialID() string {
	if input := c.APIKeyInput(); input != nil {
		return strings.TrimSpace(input.CredentialID)
	}
	return ""
}

func (c *ConnectCommand) SurfaceModels() []*SurfaceModelInput {
	if input := c.APIKeyInput(); input != nil {
		return input.SurfaceModels
	}
	return nil
}

func (c *ConnectCommand) IsSurfaceAPIKey() bool {
	return c.SurfaceID() != ""
}

func (c *ConnectCommand) IsCustomAPIKey() bool {
	return c.SurfaceID() == registry.SurfaceIDCustomAPIKey
}

func (c *ConnectCommand) ValidateAPIKey() error {
	material := c.APIKeyInput()
	if material == nil || strings.TrimSpace(material.CredentialID) == "" {
		return domainerror.NewValidation("platformk8s/providerconnect: credential_id is required")
	}
	if c.IsCustomAPIKey() {
		return c.validateCustomAPIKey()
	}
	if c.IsSurfaceAPIKey() {
		return c.validateSurfaceAPIKey()
	}
	return domainerror.NewValidation("platformk8s/providerconnect: surface_id is required for API key connect")
}

func (c *ConnectCommand) ValidateCLIOAuth() error {
	if c.SurfaceID() == "" {
		return domainerror.NewValidation("platformk8s/providerconnect: surface_id is required for CLI OAuth")
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

func (c *ConnectCommand) validateSurfaceAPIKey() error {
	material := c.APIKeyInput()
	if material.Protocol != apiprotocolv1.Protocol_PROTOCOL_UNSPECIFIED {
		return domainerror.NewValidation("platformk8s/providerconnect: provider surface API key connect does not accept protocol")
	}
	return nil
}
