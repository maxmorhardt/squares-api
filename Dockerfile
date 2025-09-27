FROM --platform=linux/arm64 alpine:latest

RUN apk add --no-cache ca-certificates

COPY squares-api .

EXPOSE 8080

CMD ["./squares-api"]
