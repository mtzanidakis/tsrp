FROM golang:1.21-alpine AS builder
RUN apk update && \
	apk add --no-cache make ca-certificates tzdata upx && \
	update-ca-certificates
WORKDIR /src
COPY . .
RUN make build-static
RUN upx --best --lzma tsrp
RUN install -d -m 1777 /tmp/tsrp-state

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /src/tsrp /bin/tsrp
COPY --from=builder /tmp/tsrp-state /var/lib/tsrp

#USER tsrp:tsrp
ENTRYPOINT ["/bin/tsrp"]
