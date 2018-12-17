# Build stage
FROM golang:1.11 as builder

LABEL maintainer="Travix"

COPY ./ /go/src/gozzmock
ENV GO111MODULE=on
WORKDIR /go/src/gozzmock
RUN go mod vendor
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -mod vendor -o gozzmock_bin .

# Run stage
FROM scratch

LABEL maintainer="Travix"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/gozzmock/gozzmock_bin .

EXPOSE 8080

ENTRYPOINT ["./gozzmock_bin"]
