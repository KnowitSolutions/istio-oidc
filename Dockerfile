FROM golang:1.14 AS builder
WORKDIR /build

RUN apt-get update && apt-get install --yes protobuf-compiler
COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go install \
    sigs.k8s.io/controller-tools/cmd/controller-gen \
    google.golang.org/protobuf/cmd/protoc-gen-go \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc

COPY . .
RUN go generate ./...
RUN go build -trimpath -tags osusergo,netgo -ldflags="-w -s" -o istio-keycloak

FROM scratch
ENV PATH /

COPY --from=builder /build/istio-keycloak .
ENTRYPOINT ["istio-keycloak"]
EXPOSE 8080 8081