package api

//go:generate controller-gen object
//go:generate protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. replication.proto
