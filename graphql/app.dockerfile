FROM golang:1.24.5-bookworm AS build

# Install build dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    make \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /go/src/github.com/sdshah09/GoCore

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY account account
COPY product product
COPY order order
COPY graphql graphql

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/app ./graphql

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/bin

# Copy binary from build stage
COPY --from=build /go/bin/app .

EXPOSE 8080

CMD ["./app"]
