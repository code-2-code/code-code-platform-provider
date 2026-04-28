package providerconnect

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
)

func TestNewConnectCommandRequiresAddMethod(t *testing.T) {
	_, err := NewConnectCommand(ConnectCommandInput{})
	if err == nil {
		t.Fatal("NewConnectCommand() error = nil, want validation error")
	}
}

func TestConnectCommandRejectsVendorAPIKeySurfaceFields(t *testing.T) {
	command, err := NewConnectCommand(ConnectCommandInput{
		AddMethod: AddMethodAPIKey,
		VendorID:  "openai",
		APIKey: &APIKeyConnectInput{
			APIKey:   "secret",
			BaseURL:  "https://api.example.com/v1",
			Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
		},
	})
	if err != nil {
		t.Fatalf("NewConnectCommand() error = %v", err)
	}
	if err := command.ValidateAPIKey(); err == nil {
		t.Fatal("ValidateAPIKey() error = nil, want validation error")
	}
}

func TestConnectCommandRequiresCustomAPIKeyFields(t *testing.T) {
	command, err := NewConnectCommand(ConnectCommandInput{
		AddMethod: AddMethodAPIKey,
		APIKey: &APIKeyConnectInput{
			APIKey: "secret",
		},
	})
	if err != nil {
		t.Fatalf("NewConnectCommand() error = %v", err)
	}
	if err := command.ValidateAPIKey(); err == nil {
		t.Fatal("ValidateAPIKey() error = nil, want validation error")
	}
}

func TestConnectCommandRequiresCLIIDForOAuth(t *testing.T) {
	command, err := NewConnectCommand(ConnectCommandInput{
		AddMethod: AddMethodCLIOAuth,
	})
	if err != nil {
		t.Fatalf("NewConnectCommand() error = %v", err)
	}
	if err := command.ValidateCLIOAuth(); err == nil {
		t.Fatal("ValidateCLIOAuth() error = nil, want validation error")
	}
}

func TestConnectCommandTrimsAPIKeyValue(t *testing.T) {
	command, err := NewConnectCommand(ConnectCommandInput{
		AddMethod: AddMethodAPIKey,
		VendorID:  "openai",
		APIKey: &APIKeyConnectInput{
			APIKey: " secret ",
		},
	})
	if err != nil {
		t.Fatalf("NewConnectCommand() error = %v", err)
	}
	if got, want := command.APIKeyValue(), "secret"; got != want {
		t.Fatalf("api_key = %q, want %q", got, want)
	}
}
