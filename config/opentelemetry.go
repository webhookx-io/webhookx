package config

type OtlpProtocol string

const (
	OtlpProtocolGRPC OtlpProtocol = "grpc"
	OtlpProtocolHTTP OtlpProtocol = "http/protobuf"
)
