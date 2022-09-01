FROM golang:alpine as builder

WORKDIR /app

RUN apk add build-base
COPY go.* .
RUN go mod download
COPY ../ ./

RUN --mount=type=cache,target=/root/.cache/go-build go build -o /out/main ./cmd/main.go

FROM alpine:latest as prod

EXPOSE 6060
COPY --from=builder /out/main /app/.env /
CMD /main -h $HOST -d "$DSN"