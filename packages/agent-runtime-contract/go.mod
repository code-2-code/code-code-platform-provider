module code-code.internal/agent-runtime-contract

go 1.26.2

require code-code.internal/go-contract v0.0.0

require (
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/grpc v1.80.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace code-code.internal/go-contract => ../../code-code-contracts/packages/go-contract
