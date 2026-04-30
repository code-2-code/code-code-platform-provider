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
			CredentialID: "credential-openai",
			BaseURL:      "https://api.example.com/v1",
			Protocol:     apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
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
			CredentialID: "credential-custom",
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

func TestConnectCommandTrimsCredentialID(t *testing.T) {
	command, err := NewConnectCommand(ConnectCommandInput{
		AddMethod: AddMethodAPIKey,
		VendorID:  "openai",
		APIKey: &APIKeyConnectInput{
			CredentialID: " credential-openai ",
		},
	})
	if err != nil {
		t.Fatalf("NewConnectCommand() error = %v", err)
	}
	if got, want := command.CredentialID(), "credential-openai"; got != want {
		t.Fatalf("credential_id = %q, want %q", got, want)
	}
}
