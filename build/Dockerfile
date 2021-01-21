FROM golang:1.15-alpine as builder
ENV GOPROXY=https://goproxy.io,direct
ENV GO111MODULE=on

WORKDIR /go/cache
ADD go.mod .
ADD go.sum .
RUN go mod download

WORKDIR /work
ADD . .
RUN GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix cgo -o app github.com/zc2638/review-bot/cmd

FROM alpine:3.6 as alpine
RUN apk add -U --no-cache ca-certificates

FROM alpine
ENV TS="Asia/Shanghai"

COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /work/app /work/app
COPY --from=builder /work/config /work/config
COPY --from=builder /work/public /work/public

WORKDIR /work
CMD ["./app"]
