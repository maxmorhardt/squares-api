FROM --platform=linux/arm64 alpine:latest

WORKDIR /app

COPY squares-api .

RUN apk add --no-cache ca-certificates && \
    chmod +x squares-api

EXPOSE 8080

CMD ["./squares-api"]
