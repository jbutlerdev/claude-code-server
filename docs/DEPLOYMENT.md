# Claude Code Server Deployment Guide

## Overview

This guide covers different deployment options for Claude Code Server, from local development to production environments.

## Prerequisites

- Docker and Docker Compose
- PostgreSQL 14+ (if not using Docker Compose)
- Git repositories to manage
- Anthropic API key for Claude Code

## Environment Variables

Create a `.env` file with the following variables:

```bash
# Required (choose one authentication method)
CLAUDE_CODE_API_KEY=your-anthropic-api-key  # OR authenticate via terminal

# Database
DATABASE_URL=postgres://user:password@host:port/database

# GitHub Integration (optional)
DEFAULT_GITHUB_TOKEN=ghp_your_token_here
GIT_USER_NAME=Claude Code Server
GIT_USER_EMAIL=claude@example.com

# Optional (with defaults)
SERVER_PORT=8080
MAX_CONCURRENT_SESSIONS=10
CLAUDE_CODE_BINARY=/usr/local/bin/claude
LOG_LEVEL=info
```

## Deployment Options

### 1. Docker Compose (Recommended)

The easiest way to deploy with a local PostgreSQL database:

```bash
# Clone the repository
git clone https://github.com/yourusername/claude-code-server.git
cd claude-code-server

# Set your API key (optional - can authenticate via terminal)
export CLAUDE_CODE_API_KEY="your-api-key"

# Start all services
docker compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f app
```

This starts:
- Claude Code Server on port 8080
- PostgreSQL database on port 5432
- Shared volume for git worktrees

### 2. Single Container with External Database

If you have an existing PostgreSQL instance:

```bash
docker run -d \
  --name claude-code-server \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/dbname" \
  -e CLAUDE_CODE_API_KEY="your-api-key" \
  -v /path/to/worktrees:/var/lib/claude-code-server/worktrees \
  claude-code-server:latest
```

### 3. Kubernetes Deployment

Example Kubernetes manifests:

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: claude-code-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: claude-code-server
  template:
    metadata:
      labels:
        app: claude-code-server
    spec:
      containers:
      - name: app
        image: claude-code-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: database-secret
              key: url
        - name: CLAUDE_CODE_API_KEY
          valueFrom:
            secretKeyRef:
              name: claude-secret
              key: api-key
        volumeMounts:
        - name: worktrees
          mountPath: /var/lib/claude-code-server/worktrees
      volumes:
      - name: worktrees
        persistentVolumeClaim:
          claimName: worktrees-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: claude-code-server
spec:
  selector:
    app: claude-code-server
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

### 4. Building from Source

For development or custom builds:

```bash
# Install dependencies
make deps

# Build the application
make build

# Set up database
export DATABASE_URL="postgres://user:pass@localhost/claude_code_server"
make migrate-up

# Run the server
./bin/claude-code-server
```

## Production Configuration

### Security

1. **Change default JWT secret:**
   ```bash
   JWT_SECRET=$(openssl rand -base64 32)
   ```

2. **Use API key authentication:**
   Create API keys in the database:
   ```sql
   INSERT INTO api_keys (key_hash, name, permissions) 
   VALUES (
     encode(sha256('your-secure-api-key'), 'hex'),
     'Production API Key',
     '["read", "write"]'
   );
   ```

3. **Enable TLS:**
   Use a reverse proxy like Nginx:
   ```nginx
   server {
       listen 443 ssl;
       server_name your-domain.com;
       
       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;
       
       location / {
           proxy_pass http://claude-code-server:8080;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
       }
   }
   ```

### Performance Tuning

1. **Database connections:**
   ```bash
   DATABASE_MAX_CONNECTIONS=50
   DATABASE_MAX_IDLE_CONNECTIONS=10
   ```

2. **Concurrent sessions:**
   ```bash
   MAX_CONCURRENT_SESSIONS=25
   ```

3. **Resource limits (Docker):**
   ```yaml
   services:
     app:
       deploy:
         resources:
           limits:
             memory: 2G
             cpus: '1.0'
           reservations:
             memory: 1G
             cpus: '0.5'
   ```

### Monitoring

1. **Health checks:**
   ```bash
   curl http://localhost:8080/api/health
   curl http://localhost:8080/api/ready
   ```

2. **Metrics endpoint:**
   ```bash
   curl http://localhost:8080/api/metrics
   ```

3. **Log aggregation:**
   ```yaml
   # docker-compose.yml
   services:
     app:
       logging:
         driver: "json-file"
         options:
           max-size: "10m"
           max-file: "3"
   ```

### Backup Strategy

1. **Database backups:**
   ```bash
   # Daily backup
   pg_dump $DATABASE_URL > backup_$(date +%Y%m%d).sql
   
   # Restore
   psql $DATABASE_URL < backup_20240115.sql
   ```

2. **Git worktrees cleanup:**
   Configure automatic cleanup:
   ```bash
   LOG_RETENTION_DAYS=7
   ```

## Scaling

### Horizontal Scaling

For multiple instances, ensure:

1. **Shared storage for worktrees:**
   ```yaml
   volumes:
     worktrees:
       driver: nfs
       driver_opts:
         share: nfs-server:/path/to/shared/storage
   ```

2. **Load balancer configuration:**
   ```nginx
   upstream claude_code_backend {
       server app1:8080;
       server app2:8080;
       server app3:8080;
   }
   
   server {
       location / {
           proxy_pass http://claude_code_backend;
       }
   }
   ```

3. **Session affinity for WebSockets:**
   ```nginx
   upstream claude_code_backend {
       ip_hash;
       server app1:8080;
       server app2:8080;
   }
   ```

### Database Scaling

1. **Connection pooling:**
   Use pgbouncer for connection pooling:
   ```bash
   DATABASE_URL=postgres://user:pass@pgbouncer:5432/claude_code_server
   ```

2. **Read replicas:**
   Configure read-only database connections for queries:
   ```bash
   DATABASE_READ_URL=postgres://user:pass@read-replica:5432/claude_code_server
   ```

## Troubleshooting

### Common Issues

1. **Permission denied on worktrees:**
   ```bash
   chown -R 1000:1000 /var/lib/claude-code-server/worktrees
   ```

2. **Database connection failures:**
   ```bash
   # Check connectivity
   nc -zv database-host 5432
   
   # Verify credentials
   psql $DATABASE_URL -c "SELECT 1;"
   ```

3. **Claude Code not found:**
   ```bash
   # Install in container
   docker exec -it claude-code-server npm install -g @anthropic-ai/claude-code
   ```

### Debugging

1. **Enable debug logging:**
   ```bash
   LOG_LEVEL=debug
   ```

2. **Container debugging:**
   ```bash
   # Access container shell
   docker exec -it claude-code-server sh
   
   # Check processes
   docker exec claude-code-server ps aux
   
   # View logs
   docker logs -f claude-code-server
   ```

3. **Database debugging:**
   ```bash
   # Connect to database
   docker exec -it claude-code-postgres psql -U claude -d claude_code_server
   
   # Check tables
   \dt
   
   # View sessions
   SELECT id, status, created_at FROM sessions ORDER BY created_at DESC LIMIT 10;
   ```

## Migration Guide

### From Development to Production

1. **Export development data:**
   ```bash
   pg_dump $DEV_DATABASE_URL > dev_export.sql
   ```

2. **Import to production:**
   ```bash
   psql $PROD_DATABASE_URL < dev_export.sql
   ```

3. **Update configuration:**
   - Change JWT_SECRET
   - Set production API keys
   - Configure TLS
   - Set up monitoring

### Version Updates

1. **Backup before update:**
   ```bash
   docker-compose exec db pg_dump -U claude claude_code_server > backup.sql
   ```

2. **Update image:**
   ```bash
   docker-compose pull
   docker-compose up -d
   ```

3. **Run migrations:**
   ```bash
   docker-compose exec app make migrate-up
   ```

## Security Considerations

1. **Network security:**
   - Use internal networks for database connections
   - Expose only necessary ports
   - Implement firewall rules

2. **Data protection:**
   - Encrypt database connections (SSL)
   - Secure API key storage
   - Regular security updates

3. **Access control:**
   - Implement API key rotation
   - Monitor access patterns
   - Use least privilege principle

4. **Git repository access:**
   - Use SSH keys for private repositories
   - Limit file system access
   - Implement worktree cleanup