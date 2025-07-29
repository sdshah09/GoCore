FROM golang:1.24.5-bookworm AS build
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    make \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /go/src/github.com/sdshah09/GoCore
COPY go.mod go.sum ./
RUN go mod download
COPY product product
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/app ./product/cmd/product

# Step 2 multi stage docker to use only binaries to decrease docker image size
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/bin

# Copy binary from build stage
COPY --from=build /go/bin/app .

EXPOSE 8081

CMD ["./app"]