package providerconnect

// ConnectShape identifies one provider connect onboarding shape.
type ConnectShape int32

const (
	ConnectShapeUnspecified   ConnectShape = 0
	ConnectShapeVendorSpecAPI ConnectShape = 1
	ConnectShapeCustomAPI     ConnectShape = 2
	ConnectShapeCLIOAuth      ConnectShape = 3
)
