FROM golang:1.10.0 AS build-env
WORKDIR /src
ADD vendor $GOPATH/src/
ADD client_ironic.go /src/
ADD ipmi_discovery.go /src/
ADD adapter $GOPATH/src/github.com/sapcc/ipmi_sd/adapter/
RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o ipmi_sd

FROM quay.io/prometheus/busybox:latest
LABEL maintainer "sapcc <stefan.hipfel@sap.com>"
WORKDIR /app
COPY --from=build-env /src/ipmi_sd /app/
ENTRYPOINT ["./ipmi_sd"]
