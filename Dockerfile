FROM golang:1.10.0 AS build-env
ADD vendor $GOPATH/src/
ADD cmd $GOPATH/src/github.com/sapcc/ipmi_sd/cmd/
ADD internal $GOPATH/src/github.com/sapcc/ipmi_sd/internal/
ADD pkg $GOPATH/src/github.com/sapcc/ipmi_sd/pkg/

WORKDIR /src

RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o ipmi_sd github.com/sapcc/ipmi_sd/cmd/discovery

FROM quay.io/prometheus/busybox:latest
LABEL maintainer "sapcc <stefan.hipfel@sap.com>"
WORKDIR /app
COPY --from=build-env /src/ipmi_sd /app/
ENTRYPOINT ["./ipmi_sd"]
