# ── Stage 1: build ──────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy dependency manifests first so Docker layer-caches module downloads
# independently of source changes.
COPY go.mod ./
RUN go mod download

# Copy source and compile a static binary (CGO disabled — no C deps).
# -ldflags="-s -w" strips debug symbols and DWARF info (~30% size reduction).
COPY src/ ./src/
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /expense-tracker \
    ./src/main.go

# ── Stage 2: minimal runtime image ──────────────────────────────────────────
# scratch has no shell, no libc, no package manager — attack surface is zero.
# The binary is fully static so it runs without any OS libraries.
FROM scratch

# Copy CA certificates from the builder so HTTPS calls (if ever added) work.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy only the compiled binary.
COPY --from=builder /expense-tracker /expense-tracker

# The data directory is created by the app at runtime via os.MkdirAll,
# but we pre-declare it as a VOLUME so docker-compose (and plain docker run)
# can mount a named volume here and keep expenses.json across restarts.
VOLUME ["/data"]

# Environment defaults (overridable at runtime via -e or docker-compose env).
ENV PORT=8080
ENV DATA_FILE=/data/expenses.json

EXPOSE 8080

ENTRYPOINT ["/expense-tracker"]
