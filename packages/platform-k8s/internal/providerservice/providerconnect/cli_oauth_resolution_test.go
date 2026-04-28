package providerconnect

import (
	"context"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestProviderConnectCLIOAuthResolutionRuntimeResolveConnect(t *testing.T) {
	runtime := newProviderConnectCLIOAuthResolutionRuntime(
		newProviderConnectSupport(
			nil,
			cliSupportReaderStub{items: map[string]*supportv1.CLI{
				"codex": testCLIResolutionPackage("OpenAI Codex", "openai", credentialv1.OAuthAuthorizationFlow_O_AUTH_AUTHORIZATION_FLOW_DEVICE),
			}},
		),
		newProviderConnectQueries(
			nil,
			nil,
			definitionReaderStub{items: map[string]*providerv1.ProviderSurface{
				"codex": testCLIOAuthSurface("codex"),
			}},
		),
	)

	command, err := NewConnectCommand(ConnectCommandInput{
		AddMethod: AddMethodCLIOAuth,
		CLIID:     "codex",
	})
	if err != nil {
		t.Fatalf("NewConnectCommand() error = %v", err)
	}
	resolved, err := runtime.ResolveConnect(context.Background(), command)
	if err != nil {
		t.Fatalf("ResolveConnect() error = %v", err)
	}
	if got, want := resolved.flow, credentialv1.OAuthAuthorizationFlow_O_AUTH_AUTHORIZATION_FLOW_DEVICE; got != want {
		t.Fatalf("flow = %v, want %v", got, want)
	}
	if got, want := resolved.target.DisplayName, "OpenAI Codex"; got != want {
		t.Fatalf("display_name = %q, want %q", got, want)
	}
	if got, want := resolved.target.VendorID, "openai"; got != want {
		t.Fatalf("vendor_id = %q, want %q", got, want)
	}
}

func TestProviderConnectCLIOAuthResolutionRuntimeResolveReauthorize(t *testing.T) {
	runtime := newProviderConnectCLIOAuthResolutionRuntime(
		newProviderConnectSupport(
			nil,
			cliSupportReaderStub{items: map[string]*supportv1.CLI{
				"codex": testCLIResolutionPackage("", "openai", credentialv1.OAuthAuthorizationFlow_O_AUTH_AUTHORIZATION_FLOW_CODE),
			}},
		),
		nil,
	)

	resolved, err := runtime.ResolveReauthorize(context.Background(), &ProviderView{
		ProviderID:           "provider-codex",
		DisplayName:          "Codex Provider",
		ProviderCredentialID: "credential-codex",
		VendorID:             "openai",
		Surfaces: []*ProviderSurfaceBindingView{{
			SurfaceID: "codex",
			Runtime:   testCLISurfaceRuntime("codex", "codex"),
		}},
	})
	if err != nil {
		t.Fatalf("ResolveReauthorize() error = %v", err)
	}
	if got, want := resolved.flow, credentialv1.OAuthAuthorizationFlow_O_AUTH_AUTHORIZATION_FLOW_CODE; got != want {
		t.Fatalf("flow = %v, want %v", got, want)
	}
	if got, want := resolved.target.TargetCredentialID, "credential-codex"; got != want {
		t.Fatalf("target_credential_id = %q, want %q", got, want)
	}
	if got, want := resolved.target.SurfaceID, "codex"; got != want {
		t.Fatalf("target surface_id = %q, want %q", got, want)
	}
}

type cliSupportReaderStub struct {
	items map[string]*supportv1.CLI
}

func (s cliSupportReaderStub) Get(_ context.Context, cliID string) (*supportv1.CLI, error) {
	return s.items[cliID], nil
}

type definitionReaderStub struct {
	items map[string]*providerv1.ProviderSurface
}

func (s definitionReaderStub) Get(_ context.Context, surfaceID string) (*providerv1.ProviderSurface, error) {
	return s.items[surfaceID], nil
}

func testCLIResolutionPackage(
	displayName string,
	vendorID string,
	flow credentialv1.OAuthAuthorizationFlow,
) *supportv1.CLI {
	return &supportv1.CLI{
		DisplayName: displayName,
		VendorId:    vendorID,
		Oauth: &supportv1.OAuthSupport{
			Flow: flow,
		},
	}
}

func testCLIOAuthSurface(surfaceID string) *providerv1.ProviderSurface {
	return testProviderSurface(
		surfaceID,
		providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_CLI,
		[]credentialv1.CredentialKind{credentialv1.CredentialKind_CREDENTIAL_KIND_OAUTH},
	)
}
