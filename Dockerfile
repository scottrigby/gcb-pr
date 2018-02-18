FROM golang:alpine AS build
WORKDIR /go/src/github.com/scottrigby/gcb-pr
ADD . .
RUN apk --no-cache add git ca-certificates && \
    go get -u github.com/golang/dep/... && \
    dep ensure -v --vendor-only && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main . && cp main /tmp/

FROM scratch
WORKDIR /webhooks/
COPY --from=build /tmp/main .
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 3016
CMD ["./main"]
