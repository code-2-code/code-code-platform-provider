package providersurfaces

import (
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	"google.golang.org/protobuf/proto"
)

func cloneSurface(surface *supportv1.Surface) *supportv1.Surface {
	if surface == nil {
		return nil
	}
	return proto.Clone(surface).(*supportv1.Surface)
}
