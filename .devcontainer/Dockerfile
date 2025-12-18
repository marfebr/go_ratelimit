# Dev container image for Go rate limiter development
FROM mcr.microsoft.com/devcontainers/go:1.25

# Install common tools if needed
RUN apt-get update \
    && apt-get install -y --no-install-recommends make git \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Ensure GOPATH/bin is available (already in base, kept for clarity)
ENV GOPATH=/go
ENV PATH="${GOPATH}/bin:/usr/local/go/bin:${PATH}"
