package providerobservability

import (
	"context"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
	"google.golang.org/grpc"
)

type ObservabilityAuthClient interface {
	GetEgressAuthPolicy(context.Context, *authv1.GetEgressAuthPolicyRequest, ...grpc.CallOption) (*authv1.GetEgressAuthPolicyResponse, error)
	ReadCredentialMaterialFields(context.Context, *authv1.ReadCredentialMaterialFieldsRequest, ...grpc.CallOption) (*authv1.ReadCredentialMaterialFieldsResponse, error)
	ResolveEgressRequestHeaders(context.Context, *authv1.ResolveEgressRequestHeadersRequest, ...grpc.CallOption) (*authv1.ResolveEgressRequestHeadersResponse, error)
}
