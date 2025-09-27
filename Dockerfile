FROM --platform=linux/arm64 alpine:latest

RUN apk add --no-cache ca-certificates

COPY ./server .

EXPOSE 8080

CMD ["./server"]
