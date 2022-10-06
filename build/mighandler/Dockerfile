# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY ../../go.mod go.mod
COPY ../../go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/mighandler/main.go main.go
COPY ../../pkg pkg/

# Build
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -o mig-handler -tags nvml main.go

FROM nvidia/cuda:11.6.2-base-ubuntu20.04
WORKDIR /
COPY --from=builder /workspace/mig-handler .
#USER 65532:65532 TODO: Run as non-root

ENTRYPOINT ["/mig-handler"]
