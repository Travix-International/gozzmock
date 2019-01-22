# Build stage
FROM golang:1.11 as builder

# Create the user and group files that will be used in the running container to
# run the process as an unprivileged user.
RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group

# Import the code from the context.
COPY ./ /src

# Set the working directory outside $GOPATH to enable the support for modules.
WORKDIR /src

# Download modules to local cache
RUN go mod download

# Set the environment variables for the go command:
# * CGO_ENABLED=0 to build a statically-linked executable
# Build the executable to `/gozzmock_bin`. Mark the build as statically linked.
RUN CGO_ENABLED=0 GOOS=linux go build -a -o /gozzmock_bin .

# Run stage
FROM scratch AS final

LABEL maintainer="Travix"

# Import the user and group files from the first stage.
COPY --from=builder /user/group /user/passwd /etc/

# Import the Certificate-Authority certificates for enabling HTTPS.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import the compiled executable from the second stage.
COPY --from=builder /gozzmock_bin /gozzmock_bin

EXPOSE 8080

# Perform any further action as an unprivileged user.
USER nobody:nobody

ENTRYPOINT ["./gozzmock_bin"]
