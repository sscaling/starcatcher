FROM alpine:latest
RUN apk add --no-cache ca-certificates
ADD main /
ENTRYPOINT ["/main"]