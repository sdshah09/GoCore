# GoCore - Microservices GraphQL API

A microservices-based GraphQL API built with Go.

## Architecture

```
Client Request
    ↓
Server (HTTP/API)
    ↓
Service (Business Logic)
    ↓
Repository (Data Access)
    ↓
Database (PostgreSQL)
```

## Project Structure

```
GoCore/
├── account/           # Account microservice
├── order/             # Order microservice  
├── product/           # Product microservice
├── graphql/           # GraphQL API gateway
└── docker-compose.yaml
```

## Quick Start

```bash
# Install dependencies
go mod tidy

# Run with Docker
docker-compose up

# Access GraphQL Playground
http://localhost:8080/playground
```

## Protobuf Generation

### Install Tools
```bash
# Install protoc compiler
brew install protobuf

# Install Go protobuf plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Generate Code
```bash
cd account
protoc \
  --go_out=pb --go_opt=paths=source_relative \     # Generate Go message types to pb/ directory with relative paths
  --go-grpc_out=pb --go-grpc_opt=paths=source_relative \  # Generate gRPC service code to pb/ directory with relative paths
  account.proto
```

## API Endpoints

- **GraphQL**: `http://localhost:8080/graphql`
- **Playground**: `http://localhost:8080/playground`

--> Uppercase first letter → Exported (public)

--> Lowercase first letter → Unexported (private to the package)

