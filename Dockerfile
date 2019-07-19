FROM golang:alpine AS builder
RUN apk --no-cache --no-progress add --virtual build-deps build-base git ca-certificates
WORKDIR /go/src/github.com/teizz/repogirl
COPY . .
RUN make clean depend repogirl

FROM scratch
COPY --from=builder /go/src/github.com/teizz/repogirl/repogirl /repogirl
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

EXPOSE 8080
CMD ["/repogirl"]
