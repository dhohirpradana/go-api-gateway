FROM golang AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o gateway .

# Runtime container
FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache ca-certificates sqlite-libs

COPY --from=builder /app/gateway .
COPY config/ ./config/

EXPOSE 8080
CMD ["./gateway"]
