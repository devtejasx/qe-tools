FROM registry.access.redhat.com/ubi9/go-toolset:9.8-1783442369 AS builder

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY main.go main.go
COPY config/ config/
COPY tools/ tools/
COPY pkg/ pkg/
COPY cmd/ cmd/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o qe-tools main.go

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8-1780378819

LABEL konflux.additional-tags="latest"

USER 65532:65532

WORKDIR /qe-tools

COPY --from=builder /opt/app-root/src/qe-tools /usr/bin/
COPY --from=builder /opt/app-root/src/config config


