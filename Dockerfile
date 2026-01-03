FROM alpine:latest

LABEL io.maxstash.image.source="https://github.com/maxmorhardt/squares-api"
LABEL io.maxstash.image.description="Squares API - Game squares management with real-time updates"
LABEL io.maxstash.image.vendor="Max Morhardt"
LABEL io.maxstash.image.licenses="MIT"

ENV GIN_MODE release

WORKDIR /app

RUN addgroup -g 1000 squares && \
    adduser -D -u 1000 -G squares squares

COPY --chown=squares:squares squares-api .

RUN apk add --no-cache ca-certificates && \
    chmod +x squares-api

USER squares

EXPOSE 8080

CMD ["./squares-api"]
