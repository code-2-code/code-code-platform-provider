package providerservice

import (
	"context"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	oauthv1 "code-code.internal/go-contract/platform/oauth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type remoteOAuthSessionService struct {
	client oauthv1.OAuthSessionServiceClient
}

func newRemoteOAuthSessionService(client oauthv1.OAuthSessionServiceClient) *remoteOAuthSessionService {
	return &remoteOAuthSessionService{client: client}
}

func (s *remoteOAuthSessionService) StartSession(ctx context.Context, request *credentialv1.OAuthAuthorizationSessionSpec) (*credentialv1.OAuthAuthorizationSessionState, error) {
	if s == nil || s.client == nil {
		return nil, status.Error(codes.Unavailable, "oauth session service is unavailable")
	}
	response, err := s.client.StartOAuthAuthorizationSession(ctx, &oauthv1.StartOAuthAuthorizationSessionRequest{
		CliId:              strings.TrimSpace(request.GetCliId()),
		Flow:               request.GetFlow(),
		TargetCredentialId: strings.TrimSpace(request.GetTargetCredentialId()),
		TargetDisplayName:  strings.TrimSpace(request.GetTargetDisplayName()),
	})
	if err != nil {
		return nil, err
	}
	return response.GetSession(), nil
}

func (s *remoteOAuthSessionService) GetSession(ctx context.Context, sessionID string) (*credentialv1.OAuthAuthorizationSessionState, error) {
	if s == nil || s.client == nil {
		return nil, status.Error(codes.Unavailable, "oauth session service is unavailable")
	}
	response, err := s.client.GetOAuthAuthorizationSession(ctx, &oauthv1.GetOAuthAuthorizationSessionRequest{
		SessionId: strings.TrimSpace(sessionID),
	})
	if err != nil {
		return nil, err
	}
	return response.GetSession(), nil
}

func (s *remoteOAuthSessionService) CancelSession(ctx context.Context, sessionID string) (*credentialv1.OAuthAuthorizationSessionState, error) {
	if s == nil || s.client == nil {
		return nil, status.Error(codes.Unavailable, "oauth session service is unavailable")
	}
	response, err := s.client.CancelOAuthAuthorizationSession(ctx, &oauthv1.CancelOAuthAuthorizationSessionRequest{
		SessionId: strings.TrimSpace(sessionID),
	})
	if err != nil {
		return nil, err
	}
	return response.GetSession(), nil
}
