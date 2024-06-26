FROM golang:1.21.10 as builder
WORKDIR /build
COPY . /build/
RUN make build

FROM alpine:3.18
WORKDIR /app
RUN apk add libc6-compat
COPY --from=builder /build/infra-websocket-gateway-service /app
RUN mkdir -p /app/logs
EXPOSE 16887
CMD [ \
    "/app/infra-websocket-gateway-service", \
    "-conf", \
    "/config/infra-websocket-gateway-service.json" \
]
