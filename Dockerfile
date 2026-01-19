FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o scoring main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/scoring .

EXPOSE 8082

ENV PORT=8082
ENV CONFIG_API_URL=https://api.conturs.com

CMD ["./scoring"]
