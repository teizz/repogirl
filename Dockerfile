FROM golang:alpine AS builder
RUN apk --no-cache --no-progress add make git ca-certificates upx
WORKDIR /go/src/github.com/teizz/repogirl
COPY . .
RUN make clean depend
RUN make repogirl && \
    upx repogirl > /dev/null

FROM scratch
COPY --from=builder /go/src/github.com/teizz/repogirl/repogirl /repogirl
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

EXPOSE 8080 8443
CMD ["/repogirl"]
