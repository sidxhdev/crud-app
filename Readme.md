# Blog Application - 3-Tier Architecture

A complete educational project demonstrating a 3-tier application architecture with React, Go, and PostgreSQL, containerized with Docker and deployable to AWS using EC2 + Docker.

## Purpose

This project is designed to teach Docker and 3-tier deployment patterns with a focus on EC2 + Docker workflow. It includes:

- Frontend: React + Vite (modern, fast client-side rendering)
- Backend: Go HTTP API (performance-oriented server-side logic)
- Database: PostgreSQL (relational data storage)

Each component has its own Dockerfile and they orchestrate together via Docker Compose. The application is ready for deployment to an EC2 instance using Docker.

## Project Structure

```
crud-app/
├── backend/              # Go language REST API
│   ├── main.go           # Blog API with CRUD endpoints
│   ├── Dockerfile        # Multi-stage Go build
│   ├── go.mod            # Module dependencies
│   └── go.sum            # Go checksums
├── frontend/             # React + Vite application
│   ├── src/              # React source components
│   ├── Dockerfile        # Node build → Nginx serve
│   ├── vite.config.js    # Development proxy configuration
│   └── package.json      # Frontend dependencies
├── docker-compose.yml    # Orchestrates db, backend, frontend
├── Readme.md             # This documentation
└── .github/              # (Optional) GitHub workflows
```

## Quick Start

### Using Docker Compose (Recommended - Local Development)

```bash
# Start all services
docker compose up -d

# Access the application
# Frontend:     http://localhost:3000
# Backend API:    http://localhost:8080
# PostgreSQL:     localhost:5433 (maps to container port 5432)

# Test the API
curl -s -X POST http://localhost:8080/api/posts -d "title=Hello" -d "content=World"
curl -s http://localhost:8080/api/posts
curl -s http://localhost:3000
```

### Without Docker (Local Development)

**Backend:**
```bash
cd backend
go mod tidy
go run main.go
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

**Database:** Ensure PostgreSQL is running with the `blog` database, user `postgres`, and password `postgres`.

## Docker Architecture

This project demonstrates multi-stage builds best practices:

### Backend Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
RUN go build -o blog-backend .

FROM alpine:3.19

USER appuser

WORKDIR /app

COPY --from=builder /app/blog-backend .

EXPOSE 8080

CMD ["./blog-backend"]
```

**Why multi-stage?** Builder stage uses `golang:1.21-alpine` with Go tools, final stage uses minimal `alpine:3.19` with only the compiled binary - resulting in smaller image size and better security.

### Frontend Dockerfile

```dockerfile
FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm install

COPY . .

RUN npm run build

FROM nginx:alpine

COPY --from=builder /app/dist ./usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
```

**Why multi-stage?** `node:20-alpine` builds the React app, then `nginx:alpine` serves the static files - smallest possible production image.

### docker-compose.yml

```yaml
version: '3.8'

services:
  db:
    image: postgres:15
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: blog
    ports:
      - "5433:5432"  # Host:Container
    volumes:
      - postgres_data:/var/lib/postgresql/data

  backend:
    build: ./backend
    depends_on:
      - db
    ports:
      - "8080:8080"

  frontend:
    build: ./frontend
    depends_on:
      - backend
    ports:
      - "3000:80"
    environment:
      VITE_API_URL: http://localhost:8080

volumes:
  postgres_data:
```

**Key concepts taught:**
- Service dependencies (`depends_on`)
- Port mapping (host:container)
- Named volumes for data persistence
- Environment variable injection

## AWS Deployment - EC2 + Docker

This pathway deploys the application to an EC2 instance using Docker directly. No ECS, no Fargate, no additional AWS services required - just EC2 and Docker.

### Step 1: Launch EC2 Instance

- Amazon Linux 2023 or Ubuntu Server
- Security groups: Allow ports:
  - 22 (SSH)
  - 80 (HTTP - frontend)
  - 5433 (PostgreSQL)
- Key pair for SSH access

### Step 2: Install Docker on EC2

**For Amazon Linux 2023:**
```bash
sudo yum install -y docker docker-compose
sudo service docker start
sudo usermod -a -G docker ec2-user
```

**For Ubuntu:**
```bash
sudo apt-get update
sudo apt-get install -y docker.io docker-compose
sudo usermod -a -G docker $USER
```

### Step 3: Pull and Run the Application

```bash
# On EC2 instance via SSH

# Option A: Pull pre-built images from Docker Hub
docker pull postgres:15                          # Database
docker pull your-dockerhub-username/crudapp-backend:latest  # Backend
docker pull your-dockerhub-username/crudapp-frontend:latest  # Frontend

# Or build on EC2 directly from this repository
cd /path/to/crud-app
docker compose up -d  # Uses local docker-compose.yml

# Verify running containers
docker compose ps
```

### Step 4: Access the Application

- **Frontend**: http://<EC2-public-ip>:3000
  - Or configure Nginx reverse proxy on port 80 for clean URL
- **Backend API**: http://<EC2-public-ip>:8080/api/posts
- **PostgreSQL**: localhost:5433 (host) → 5432 (container)
  - Or migrate to Amazon RDS for production reliability

**Example access:**
```bash
# Test frontend
curl -s http://<EC2-public-ip>:3000

# Test backend API
curl -s http://<EC2-public-ip>:8080/api/posts

# Create a new post
curl -s -X POST -d "title=My Post" -d "content=Hello" http://<EC2-public-ip>:8080/api/posts
```

### Step 5: (Optional) Production Enhancements

- **Domain & HTTPS**: Purchase domain through Route 53, configure ACM certificate, set up Nginx reverse proxy on EC2
- **Load Balancer**: Add Application Load Balancer (ALB) in front of EC2 for high availability
- **Database**: Migrate from container PostgreSQL to Amazon RDS for automated backups and reliability
- **Monitoring**: Use CloudWatch basic metrics (included in free tier)
- **Auto Scaling**: Configure EC2 auto-scaling group if needed

## EC2 vs Other Approaches

| Factor | EC2 + Docker (This Guide) |
|--------|---------------------------|
| **Cost** | Just EC2 instance cost (lowest option) |
| **Complexity** | Simple - just Docker on EC2 |
| **Scalability** | Manual EC2 scaling |
| **Management** | You manage the Docker containers |
| **Best for** | Learning Docker, small projects, cost efficiency |
| **Production** | Good for small-mid apps, requires ops attention |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/posts | Retrieve all posts |
| POST | /api/posts | Create new post (form data: title, content) |
| GET | /api/posts/{id} | Retrieve single post by ID |
| PUT | /api/posts/{id} | Update post by ID |
| DELETE | /api/posts/{id} | Delete post by ID |

**Example - Create post:**
```bash
curl -X POST http://localhost:8080/api/posts \
  -d "title=My First Post" \
  -d "content=Hello World"
```

**Example - Response:**
```json
{"id": "1"}
```

**Example - List posts:**
```json
{"posts": [{"id": "1", "title": "My First Post", "content": "Hello World"}]}
```

## Learning Outcomes

After completing this tutorial, you will be able to:

- Design and implement a 3-tier application architecture
- Write multi-stage Dockerfiles for optimized image sizes
- Orchestrate multi-container applications with Docker Compose
- Deploy Docker applications to AWS EC2
- Configure service dependencies, networks, and volumes
- troubleshoot common Docker and deployment issues
- Plan EC2-based hosting considerations

## Troubleshooting

### Common Issues

**Port already in use:**
```bash
docker compose down -v
# Kill processes on ports: lsof -i :5433 :8080 :3000
docker compose up -d
```

**Backend can't connect to database:**
- Ensure `db` service is running: `docker compose ps`
- Check network connectivity
- Verify DATABASE_URL matches docker-compose environment variables

**Frontend can't reach API:**
- Check VITE_API_URL environment variable
- Ensure backend service is healthy: `docker compose logs backend`
- Verify proxy configuration in vite.config.js

**Out of memory errors:**
- Increase EC2 instance size
- Monitor with: `docker stats --no-stream`

**Cannot connect to EC2 via SSH:**
- Check security groups allow port 22
- Verify key pair is correct
- Ensure instance is running

## Production Checklist (EC2)

- [ ] Use Amazon RDS PostgreSQL instead of container database
- [ ] Configure Application Load Balancer (ALB) in front of EC2
- [ ] Set up SSL/TLS certificates (ACM + Route 53)
- [ ] Implement health checks in Docker Compose
- [ ] Add environment variable validation
- [ ] Configure logging to CloudWatch
- [ ] Set up automated RDS backups
- [ ] Implement CI/CD pipeline for Docker image updates
- [ ] Add rate limiting and security headers
- [ ] Monitor with CloudWatch Dashboards

## Next Steps

1. **Experiment**: Modify the Go API, add new endpoints, change the React UI
2. **Scale**: Add a second EC2 instance, configure manual load balancing
3. **Secure**: Implement authentication, add HTTPS via Nginx/ACM
4. **Optimize**: Analyze image sizes, improve database queries
5. **Automate**: Set up simple deployment script for future updates

## Tutorial Reference

For step-by-step local setup instructions, see the companion tutorial:
- `DOCKER-3TIER-TUTORIAL.md` - Complete guide from project creation to local running

---

Built with for teaching Docker and cloud deployment

*This project demonstrates modern full-stack development practices with containerization at the core, focused on EC2 + Docker deployment pathway.*