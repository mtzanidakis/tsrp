FROM golang:1.24.3-alpine AS builder
RUN apk update && \
	apk add --no-cache make ca-certificates tzdata upx && \
	update-ca-certificates
WORKDIR /src
COPY . .
RUN make build-static
RUN upx --best --lzma tsrp

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /src/tsrp /bin/tsrp
ENTRYPOINT ["/bin/tsrp"]
