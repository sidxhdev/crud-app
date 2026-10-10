# Blog Application - 3-Tier Architecture

A complete educational project demonstrating a 3-tier application architecture with React, Go, and PostgreSQL, containerized with Docker and deployable to AWS.

## 🎯 Purpose

This project is designed to teach **Docker** and **3-tier deployment** patterns. It includes:

- **Frontend**: React + Vite (modern, fast client-side rendering)
- **Backend**: Go HTTP API (performance-oriented server-side logic)
- **Database**: PostgreSQL (relational data storage)

Each component has its own Dockerfile and they orchestrate together via Docker Compose. The application is ready for AWS deployment using ECR + EC2 or ECS Fargate.

## 📁 Project Structure

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

## ⚡ Quick Start

### Using Docker Compose (Recommended)

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

### Without Docker

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

**Database:** Ensure PostgreSQL is running with `blog` database, user `postgres`, password `postgres`.

## 🐳 Docker Architecture

This project demonstrates **multi-stage builds** best practices:

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

## 🚀 AWS Deployment

### Option 1: ECR + EC2 (Recommended for learning)

**Step 1: Push images to Amazon ECR**

```bash
# Authenticate
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <account-id>.dkr.ecr.us-east-1.amazonaws.com

# Tag and push frontend
docker tag crudapp-frontend:latest <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-frontend:latest
docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-frontend:latest

# Tag and push backend
docker tag crudapp-backend:latest <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-backend:latest
docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-backend:latest

# PostgreSQL uses official image from Docker Hub
```

**Step 2: Launch EC2 Instance**

- Amazon Linux 2023 or Ubuntu Server
- Security groups: Allow ports 22 (SSH), 80 (HTTP), 5433 (Postgres)
- Key pair for SSH access

**Step 3: Deploy on EC2**

```bash
# On EC2 instance
sudo yum install -y docker docker-compose  # Amazon Linux
sudo service docker start
sudo usermod -a -G docker ec2-user

# Pull and run
docker pull <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-frontend:latest
docker pull <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-backend:latest

# Using docker compose
cd /path/to/project
docker compose up -d
```

**Step 4: Access Application**

- Frontend: http://<EC2-public-ip>/  (or via ALB)
- Backend API: http://<EC2-public-ip>:8080/api/posts

### Option 2: ECS Fargate (Managed)

1. Create ECS cluster
2. Register task definition using ECR images
3. Deploy service with Fargate launch type
4. Set up ALB with target groups

### Option 3: Elastic Beanstalk

- **Frontend**: S3 + CloudFront static website
- **Backend**: Docker container on EB
- **Database**: Amazon RDS PostgreSQL

## 🔌 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/posts` | Retrieve all posts |
| `POST` | `/api/posts` | Create new post (form data: title, content) |
| `GET` | `/api/posts/{id}` | Retrieve single post by ID |
| `PUT` | `/api/posts/{id}` | Update post by ID |
| `DELETE` | `/api/posts/{id}` | Delete post by ID |

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

## 📚 Learning Objectives

Through this project, you will learn:

1. **Docker fundamentals**: Images, containers, networks, volumes
2. **Multi-stage builds**: Reducing image size, separation of concerns
3. **Docker Compose**: Multi-container orchestration, service dependencies
4. **3-tier architecture**: Separation of presentation, logic, and data
5. **Go backend development**: REST API, PostgreSQL integration
6. **React frontend development**: Vite build, component architecture
7. **AWS deployment**: ECR, EC2, ECS Fargate, Elastic Beanstalk
8. **Environment configuration**: DATABASE_URL, VITE_API_URL patterns

## 🐛 Troubleshooting

### Common Issues

**Port already in use:**
```bash
docker compose down -v
# Kill processes on ports: lsof -i :5433 :8080 :3000
docker compose up -d
```

**Backend can't connect to database:**
- Ensure `db` service is running: `docker compose ps`
- Check network: `docker network inspect crudapp_default`
- Verify DATABASE_URL matches docker-compose environment variables

**Frontend can't reach API:**
- Check VITE_API_URL environment variable
- Ensure backend service is healthy: `docker compose logs backend`
- Verify proxy configuration in vite.config.js

**Out of memory errors:**
- Increase EC2 instance size
- Monitor with: `docker stats --no-stream`

## 📦 Production Checklist

- [ ] Use RDS PostgreSQL instead of container database
- [ ] Configure Application Load Balancer (ALB)
- [ ] Set up SSL/TLS certificates (ACM + Route 53)
- [ ] Implement health checks in Docker Compose
- [ ] Add environment variable validation
- [ ] Configure logging to CloudWatch
- [ ] Set up automated backups for database
- [ ] Implement CI/CD pipeline for Docker image updates
- [ ] Add rate limiting and security headers
- [ ] Monitor with CloudWatch Dashboards

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add some amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

**Built with ❤️ for teaching Docker and cloud deployment**

*This project demonstrates modern full-stack development practices with containerization at the core.*