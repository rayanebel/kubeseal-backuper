ARG GO_VERSION=1.13.5
FROM golang:${GO_VERSION}-alpine AS builder

LABEL author="Rayane BELLAZAAR"

RUN apk add --no-cache ca-certificates

WORKDIR /src

COPY ./ ./

RUN go mod download && go build -o kubeseal-backuper

FROM golang:${GO_VERSION}-alpine AS final

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/kubeseal-backuper /app/

USER 1001

ENTRYPOINT ["./kubeseal-backuper"]