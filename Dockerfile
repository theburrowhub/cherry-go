# Dockerfile for cherry-go
# This Dockerfile is used by GoReleaser to build multi-arch images
# The binary is copied by GoReleaser during the build process

FROM alpine:3.19

# Install git (required for cherry-go to work with repositories)
# and ca-certificates (required for HTTPS connections)
RUN apk add --no-cache git ca-certificates

# Copy the binary (GoReleaser will copy the correct binary for each architecture)
COPY cherry-go /usr/local/bin/cherry-go

# Set working directory for mounted volumes
WORKDIR /workspace

# Run cherry-go as the entrypoint
ENTRYPOINT ["cherry-go"]

