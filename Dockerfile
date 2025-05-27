# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./

COPY *.go ./

RUN go build -v -o /prusa_proxy

FROM alpine:latest

COPY --from=builder /prusa_proxy .

EXPOSE 31100

ENTRYPOINT ["/prusa_proxy"]