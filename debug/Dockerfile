FROM golang:1.14 AS builder
WORKDIR /build

RUN apt-get update && apt-get install --yes protobuf-compiler
RUN go get github.com/go-delve/delve/cmd/dlv
COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go install \
    sigs.k8s.io/controller-tools/cmd/controller-gen \
    google.golang.org/protobuf/cmd/protoc-gen-go \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc

COPY . .
RUN go generate ./...
RUN go build -trimpath -tags osusergo,netgo -gcflags="all=-N -l" -o istio-oidc

FROM debian:buster
WORKDIR /app
ENV PATH /app:"$PATH"

COPY --from=builder /go/bin/dlv .
COPY --from=builder /build/istio-oidc .
CMD ["dlv", "--listen=:2345", "--headless", "--api-version=2", "exec", "istio-oidc"]
EXPOSE 8080 8081 2345