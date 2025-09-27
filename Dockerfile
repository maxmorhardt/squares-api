FROM --platform=linux/arm64 golang:alpine

RUN apk add --no-cache ca-certificates

COPY squares-api .

EXPOSE 8080

CMD ["./squares-api"]
