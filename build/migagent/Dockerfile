# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/migagent/migagent.go migagent.go
COPY pkg pkg/
COPY internal internal/

# Build
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -o migagent -tags nvml migagent.go

FROM nvidia/cuda:11.6.2-base-ubuntu20.04
WORKDIR /
COPY --from=builder /workspace/migagent .

ENV NVIDIA_DISABLE_REQUIRE="true"
ENV NVIDIA_VISIBLE_DEVICES=all
ENV NVIDIA_MIG_CONFIG_DEVICES=all
ENV NVIDIA_DRIVER_CAPABILITIES=utility

ENTRYPOINT ["/migagent"]
