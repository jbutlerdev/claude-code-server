FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o claude-code-server ./cmd/server

FROM alpine:latest

RUN apk add --no-cache ca-certificates git openssh-client postgresql-client util-linux nodejs npm bash

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser -s /bin/bash appuser

# Install Claude Code directly
RUN npm install -g @anthropic-ai/claude-code

# Fix shebang for BusyBox compatibility
RUN sed -i '1s|#!/usr/bin/env -S node --no-warnings --enable-source-maps|#!/usr/bin/env node|' /usr/local/bin/claude

# Copy WASM file to the same directory as claude binary
RUN cp /usr/local/lib/node_modules/@anthropic-ai/claude-code/yoga.wasm /usr/local/bin/

WORKDIR /app

COPY --from=builder /app/claude-code-server .
COPY --chown=appuser:appuser templates ./templates

RUN mkdir -p /var/lib/claude-code-server/worktrees && \
    chown -R appuser:appuser /var/lib/claude-code-server

# Create basic bash profile for appuser
RUN echo 'export PATH="/usr/local/bin:$PATH"' > /home/appuser/.bashrc && \
    echo 'export SHELL=/bin/bash' >> /home/appuser/.bashrc && \
    echo 'export HOME=/home/appuser' >> /home/appuser/.bashrc && \
    chown appuser:appuser /home/appuser/.bashrc

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

CMD ["./claude-code-server"]