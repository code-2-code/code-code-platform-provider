package providerconnect

// AddMethod identifies one provider connect onboarding path.
type AddMethod int32

const (
	AddMethodUnspecified AddMethod = 0
	AddMethodAPIKey      AddMethod = 1
	AddMethodCLIOAuth    AddMethod = 2
)
