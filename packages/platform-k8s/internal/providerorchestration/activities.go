package providerorchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"code-code.internal/go-contract/domainerror"
	authv1 "code-code.internal/go-contract/platform/auth/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	"code-code.internal/platform-k8s/internal/providerconnect"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Activities struct {
	auth     authv1.AuthServiceClient
	provider providerservicev1.ProviderServiceClient
	connect  *providerconnect.Service
}

func (a *Activities) ConnectAPIKeyProvider(ctx context.Context, input APIKeyConnectWorkflowInput) (*managementv1.ConnectProviderResponse, error) {
	if a == nil || a.connect == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider connect service is nil")
	}
	command, err := providerconnect.NewConnectCommand(providerconnect.ConnectCommandInput{
		AddMethod:   providerconnect.AddMethodAPIKey,
		DisplayName: input.DisplayName,
		SurfaceID:   input.SurfaceID,
		APIKey: &providerconnect.APIKeyConnectInput{
			CredentialID:  input.CredentialID,
			BaseURL:       input.BaseURL,
			Protocol:      input.Protocol,
			SurfaceModels: providerConnectSurfaceModels(input.SurfaceID, input.Models),
		},
	})
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError("build api key connect command", "ValidationError", err)
	}
	result, err := a.connect.Connect(ctx, command)
	if err != nil {
		return nil, activityError(err)
	}
	return managementConnectResponseFromResult(result), nil
}

func (a *Activities) ConnectCLIOAuthProvider(ctx context.Context, input CLIOAuthConnectWorkflowInput) (*managementv1.ConnectProviderResponse, error) {
	if a == nil || a.connect == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider connect service is nil")
	}
	command, err := providerconnect.NewConnectCommand(providerconnect.ConnectCommandInput{
		AddMethod:   providerconnect.AddMethodCLIOAuth,
		DisplayName: input.DisplayName,
		SurfaceID:   input.SurfaceID,
	})
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError("build cli oauth connect command", "ValidationError", err)
	}
	result, err := a.connect.Connect(ctx, command)
	if err != nil {
		return nil, activityError(err)
	}
	return managementConnectResponseFromResult(result), nil
}

func (a *Activities) ReauthorizeProvider(ctx context.Context, input CLIOAuthReauthorizationWorkflowInput) (*managementv1.UpdateProviderAuthenticationResponse, error) {
	if a == nil || a.connect == nil || a.provider == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider connect runtime is incomplete")
	}
	provider, err := a.GetProvider(ctx, input.ProviderID)
	if err != nil {
		return nil, activityError(err)
	}
	session, err := a.connect.Reauthorize(ctx, providerConnectProviderFromManagement(provider))
	if err != nil {
		return nil, activityError(err)
	}
	return &managementv1.UpdateProviderAuthenticationResponse{
		Outcome: &managementv1.UpdateProviderAuthenticationResponse_Session{
			Session: managementSessionViewFromProviderConnect(session),
		},
	}, nil
}

func (a *Activities) GetProvider(ctx context.Context, providerID string) (*managementv1.ProviderView, error) {
	if a == nil || a.provider == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider client is nil")
	}
	response, err := a.provider.ListProviders(ctx, &providerservicev1.ListProvidersRequest{})
	if err != nil {
		return nil, activityError(err)
	}
	providerID = strings.TrimSpace(providerID)
	for _, item := range response.GetItems() {
		if strings.TrimSpace(item.GetProviderId()) != providerID {
			continue
		}
		out := &managementv1.ProviderView{}
		if err := transcodeProto(item, out); err != nil {
			return nil, temporal.NewNonRetryableApplicationError("transcode provider view", "TranscodeError", err)
		}
		return out, nil
	}
	return nil, temporal.NewNonRetryableApplicationError("provider not found", "ProviderNotFound", nil, providerID)
}

func (a *Activities) GetProviderConnectSession(ctx context.Context, sessionID string) (*managementv1.GetProviderConnectSessionResponse, error) {
	if a == nil || a.connect == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider connect service is nil")
	}
	session, err := a.connect.GetSession(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, activityError(err)
	}
	return &managementv1.GetProviderConnectSessionResponse{
		Session: managementSessionViewFromProviderConnect(session),
	}, nil
}

func (a *Activities) DeleteCredential(ctx context.Context, credentialID string) (string, error) {
	if a == nil || a.auth == nil {
		return "", fmt.Errorf("platformk8s/providerorchestration: auth client is nil")
	}
	if strings.TrimSpace(credentialID) == "" {
		return "", nil
	}
	_, err := a.auth.DeleteCredential(ctx, &authv1.DeleteCredentialRequest{CredentialId: strings.TrimSpace(credentialID)})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "not-found", nil
		}
		return "", activityError(err)
	}
	return "deleted", nil
}

func activityError(err error) error {
	if err == nil {
		return nil
	}
	var validation *domainerror.ValidationError
	var notFound *domainerror.NotFoundError
	var alreadyExists *domainerror.AlreadyExistsError
	var referenceConflict *domainerror.ReferenceConflictError
	if errors.As(err, &validation) || errors.As(err, &notFound) || errors.As(err, &alreadyExists) || errors.As(err, &referenceConflict) {
		return temporal.NewNonRetryableApplicationError(err.Error(), "DomainNonRetryable", err)
	}
	switch status.Code(err) {
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.FailedPrecondition, codes.PermissionDenied, codes.Unauthenticated:
		return temporal.NewNonRetryableApplicationError(status.Convert(err).Message(), "GRPCNonRetryable", err, status.Code(err).String())
	default:
		return err
	}
}
