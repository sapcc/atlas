FROM keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/golang:1.20-alpine3.18 as builder
WORKDIR /go/src/github.com/sapcc/atlas
RUN apk add --no-cache make
COPY . .
ARG VERSION
RUN make all

FROM keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/alpine:3.18
LABEL maintainer="Stefan Hipfel <stefan.hipfel@sap.com>"
LABEL source_repository="https://github.com/sapcc/atlas"

RUN apk add --no-cache curl
RUN curl -Lo /bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.2/dumb-init_1.2.2_amd64 \
	&& chmod +x /bin/dumb-init \
	&& dumb-init -V
COPY --from=builder /go/src/github.com/sapcc/atlas/bin/linux/atlas /usr/local/bin/
ENTRYPOINT ["dumb-init", "--"]
CMD ["atlas"]
