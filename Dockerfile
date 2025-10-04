FROM --platform=linux/arm64 alpine:latest

COPY squares-api .
COPY .env .

RUN apk add --no-cache ca-certificates && \
    chmod +x squares-api

EXPOSE 8080

CMD ["sh", "-c", "./squares-api"]
