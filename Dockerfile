#build stage
FROM golang:1.20 AS builder

WORKDIR /app
COPY . .
#install dependencies
RUN go mod download
#compile
RUN CGO_ENABLED=0 GOOS=linux go build -o ./bot

#main stage
FROM alpine:latest

WORKDIR /bot
COPY --from=builder /app/bot ./wbdBot
COPY --from=builder /app/config.yaml ./config.yaml
CMD ./wbdBot
