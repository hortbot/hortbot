FROM golang:alpine as builder

RUN apk update && apk add --no-cache git ca-certificates tzdata
RUN adduser -D -g '' appuser

WORKDIR /src

COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./ ./

ARG version
RUN CGO_ENABLED=0 go build -v -ldflags="-w -s -X github.com/hortbot/hortbot/internal/version.version=${version}" -o /hortbot github.com/hortbot/hortbot

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /hortbot /hortbot
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

USER appuser
ENTRYPOINT [ "/hortbot" ]
