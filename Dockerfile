FROM golang:1.24.5-alpine as builder

WORKDIR /app

RUN apk add --no-cache make

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ENV CGO_ENABLED=0

RUN go build -ldflags="-w -s" -o webhook ./webhook

FROM alpine:latest as prod

WORKDIR /app

RUN addgroup -S appuser && adduser -S appuser -G appuser

COPY --from=builder /app/webhook .

USER appuser

CMD ["./webhook"]
