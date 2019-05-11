ARG GO_VERSION=1.12
FROM golang:${GO_VERSION}-alpine AS builder

RUN mkdir /user && \
  echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
  echo 'nobody:x:65534:' > /user/group

RUN apk add -U --no-cache ca-certificates git && update-ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -installsuffix 'static' -o /app ./cmd/device-locator/main.go


FROM scratch AS final

COPY --from=builder /user/group /user/passwd /etc/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app
COPY --from=builder /src/fmipmobile.crt /app ./

USER nobody:nobody

ENTRYPOINT ["./app"]
