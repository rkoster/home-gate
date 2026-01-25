# Build stage (with Elm and frontend build)
FROM golang:1.24 AS builder

WORKDIR /app

# Install Node.js, npm, Elm, and uglify-js
RUN apt-get update \
 && apt-get install -y curl gnupg \
 && curl -fsSL https://deb.nodesource.com/setup_18.x | bash - \
 && apt-get install -y nodejs \
 && npm install -g elm uglify-js \
 && rm -rf /var/lib/apt/lists/*

# Copy go mod, go.sum, and get deps
COPY go.mod go.sum ./
RUN go mod download

# Copy in the whole source and build script
COPY . .

# Ensure build.sh is executable
RUN chmod +x ./build.sh

# Build everything (Elm frontend, Go binary, minified JS, etc)
RUN ./build.sh

# Final minimal image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary and web assets from builder stage
COPY --from=builder /app/home-gate .
COPY --from=builder /app/web ./web

ENTRYPOINT ["./home-gate"]
CMD ["monitor"]
