FROM --platform=linux/arm64 alpine:latest

ENV GIN_MODE release

COPY squares-api .

RUN apk add --no-cache ca-certificates && \
    chmod +x squares-api

EXPOSE 8080

CMD ["sh", "-c", "./squares-api"]
