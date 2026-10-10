# Docker & 3-Tier Deployment Tutorial
## Blog Application with React, Go, and PostgreSQL

## Introduction

This tutorial guides you through building, containerizing, and deploying a full-stack blog application using a **3-tier architecture**. By the end of this guide, you'll understand how to:

1. Build a React frontend with Vite
2. Create a Go language REST API
3. Configure PostgreSQL as the database
4. Containerize each component with Docker
5. Orchestrate services with Docker Compose
6. Deploy to AWS using ECR + EC2

## Architecture Overview

A 3-tier application separates concerns into three distinct layers:

```
┌─────────────────┐      ┌──────────────────────┐      ┌─────────────────┐
│   Presentation  │      │    Business Logic    │    │   Data Layer    │
│   (Frontend)    │──→──▶│   (Backend API)      │──→──│   (Database)    │
│   React+Vite    │      │   Go HTTP Server     │      │   PostgreSQL    │
└─────────────────┘      └──────────────────────┘      └─────────────────┘
```

## Prerequisites

Before starting, ensure you have:

- **Docker** installed and running (Docker Desktop or engine)
- **Docker Compose** v2 (included with Docker Desktop)
- **AWS account** (for deployment section)
- **Basic knowledge** of HTML/CSS/Go/React
- **text editor** (VS Code, vim, etc.)

## Step 1: Initialize the Project

Create a new directory and initialize the structure:

```bash
mkdir crud-app
cd crud-app

# Create backend directory
mkdir backend
cd backend

# Initialize Go module
go mod init blog-backend

# Create go.mod with PostgreSQL driver
cat > go.mod << 'EOF'
module blog-backend

go 1.21

require github.com/lib/pq v1.10.9
EOF

# Create main.go with blog API
cat > main.go << 'GOEOF'
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@db:5432/blog?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to database!")

	http.HandleFunc("/api/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rows, err := db.Query("SELECT id, title, content FROM posts")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			var posts []map[string]string
			for rows.Next() {
				var id, title, content string
				rows.Scan(&id, &title, &content)
				posts = append(posts, map[string]string{"id": id, "title": title, "content": content})
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"posts": %v}`, posts)

		case http.MethodPost:
			title := r.FormValue("title")
			content := r.FormValue("content")
			if title == "" || content == "" {
				http.Error(w, "title and content required", http.StatusBadRequest)
				return
			}
			err := db.QueryRow("INSERT INTO posts (title, content) VALUES ($1, $2) RETURNING id", title, content).Scan(&id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id": "%s"}`, id)
		}
	})

	http.HandleFunc("/api/posts/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/api/posts/"):]

		switch r.Method {
		case http.MethodGet:
			row := db.QueryRow("SELECT id, title, content FROM posts WHERE id = $1", id)
			var id2, title, content string
			if err := row.Scan(&id2, &title, &content); err != nil {
				http.Error(w, "post not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id": "%s", "title": "%s", "content": "%s"}`, id2, title, content)

		case http.MethodPut:
			title := r.FormValue("title")
			content := r.FormValue("content")
			_, err := db.Exec("UPDATE posts SET title = $1, content = $2 WHERE id = $3", title, content, id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		case http.MethodDelete:
			_, err := db.Exec("DELETE FROM posts WHERE id = $1", id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
})
GOEOF

cd ..

# Create frontend directory
npm create vite@latest frontend -- --template react-react
cd frontend
npm install
npm install @vitejs/plugin-react

# Create vite.config.js
cat > vite.config.js << 'VITEEOF'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
    },
  },
})
VITEEOF
```

## Step 2: Create Dockerfiles

### Backend Dockerfile

```dockerfile
# syntax=docker/dockerfile:1
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

**Why multi-stage builds?**

- **Builder stage**: `golang:1.21-alpine` contains Go compiler, dependencies
- **Final stage**: `alpine:3.19` is minimal (5MB) with only the compiled binary
- **Result**: Smaller image, better security (no build tools in production)

### Frontend Dockerfile

```dockerfile
# syntax=docker/dockerfile:1
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

**Build process:**

1. `node:20-alpine` installs NPM dependencies and builds React app
2. `nginx:alpine` serves static files - production-ready web server
3. Final image only contains built assets + Nginx - ~20MB vs ~1GB+

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
      - "5433:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  backend:
    build: ./backend
    depends_on:
      - db
    environment:
      DATABASE_URL: postgres://postgres:postgres@db:5432/blog?sslmode=disable
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

**Key compose concepts:**

- `depends_on`: Service startup order (db → backend → frontend)
- `ports`: Host:Container port mapping
- `environment`: Pass environment variables to containers
- `volumes`: Persist database data (survives container restarts)

## Step 3: Run Locally

```bash
# From the project root
docker compose up -d

# Wait a few seconds, then test:
curl -s http://localhost:3000        # Frontend should load
curl -s http://localhost:8080/api/posts  # Should show empty array
curl -s -X POST -d "title=Test" -d "content=Hello" http://localhost:8080/api/posts
curl -s http://localhost:8080/api/posts  # Should show the new post
```

## Step 4: AWS Deployment (ECR + EC2)

### Prerequisites

- AWS CLI configured: `aws configure`
- ECR repository created in AWS Console
- EC2 instance launched (Amazon Linux 2023)

### Push Images to ECR

```bash
# Authenticate Docker with ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <account-id>.dkr.ecr.us-east-1.amazonaws.com

# Tag images
docker tag crudapp-frontend:latest <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-frontend:latest
docker tag crudapp-backend:latest <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-backend:latest

# Push to ECR
docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-frontend:latest
docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-backend:latest

# PostgreSQL: Use official image from Docker Hub (no need to push)
```

### Deploy to EC2

```bash
# On EC2 instance (via SSH):
sudo yum update -y
sudo yum install -y docker docker-compose
sudo service docker start
sudo usermod -a -G docker ec2-user

# Pull images from ECR
docker pull <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-frontend:latest
docker pull <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-backend:latest

# Or use docker compose with ECR images
cat > docker-compose.yml << 'EOF'
services:
  db:
    image: postgres:15
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: blog
    ports:
      - "5433:5432"
    volumes:
      - eb-postgres-data:/var/lib/postgresql/data

  backend:
    image: <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-backend:latest
    depends_on:
      - db
    ports:
      - "8080:8080"

  frontend:
    image: <account-id>.dkr.ecr.us-east-1.amazonaws.com/crudapp-frontend:latest
    depends_on:
      - backend
    ports:
      - "80:80"
    environment:
      VITE_API_URL: http://localhost:8080
volumes:
  eb-postgres-data:
EOF

docker compose up -d
```

### Production Enhancements

1. **Use RDS instead of container database**
   - More reliable, automated backups, easier scaling
   - Replace `db` service with RDS connection string

2. **Application Load Balancer (ALB)**
   - Distribute traffic across multiple EC2 instances
   - Target groups for frontend (port 80) and backend (port 8080)
   - SSL termination with ACM certificates

3. **Auto Scaling**
   - Scale EC2 instances based on CPU utilization
   - Keep application available during traffic spikes

4. **Monitoring & Logging**
   - CloudWatch metrics and logs
   - Docker logging drivers (awslogs, fluentd)
   - Custom dashboards for application health

5. **Security Best Practices**
   - Security groups restricting port access
   - IAM roles for Docker ECR access
   - Secrets management (AWS Secrets Manager vs environment variables)
   - Regular image vulnerability scanning

## Step 5: CI/CD Pipeline (Optional)

### GitHub Actions Example

```yaml
name: Build and Deploy

on:
  push:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Login to ECR
        id: ecr-login
        uses: aws-actions/amazon-ecr-login@v1
        with:
          region: us-east-1

      - name: Build and push backend
        run: |
          docker build -t blog-backend ./backend
          docker tag blog-backend <account-id>.dkr.ecr.us-east-1.amazonaws.com/blog-backend:latest
          docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/blog-backend:latest

      - name: Build and push frontend
        run: |
          docker build -t blog-frontend ./frontend
          docker tag blog-frontend <account-id>.dkr.ecr.us-east-1.amazonaws.com/blog-frontend:latest
          docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/blog-frontend:latest
```

## Troubleshooting Guide

### Common Issues & Solutions

| Problem | Cause | Solution |
|---------|-------|----------|
| `connection refused` to backend | Backend not running | `docker compose ps`, `docker compose logs backend` |
| Frontend shows 404 | Nginx serving wrong content | Rebuild: `docker compose up --build` |
| PostgreSQL won't start | Port 5433 conflict | `lsof -i :5433`, change compose port mapping |
| CORS errors | Missing headers | Add CORS middleware to Go backend, or use proxy config |
| Memory usage high | Large Docker images | Use multi-stage builds, cleanup: `docker system prune -a` |
| Environment variables not loaded | Docker Compose config issue | Verify `.env` file or compose `environment:` section |

### Debug Commands

```bash
# Check running containers
docker compose ps

# View container logs
docker compose logs [service-name]

# Execute into running container
docker exec -it [container-name] sh

# Inspect network configuration
docker network inspect crudapp_default

# Check resource usage
docker stats --no-stream
```

## Learning Outcomes

After completing this tutorial, you will be able to:

- ✅ Design and implement a 3-tier application architecture
- ✅ Write multi-stage Dockerfiles for optimized image sizes
- ✅ Orchestrate multi-container applications with Docker Compose
- ✅ Configure service dependencies, networks, and volumes
- ✅ Deploy containerized applications to AWS ECR + EC2
- ✅ troubleshoot common Docker and deployment issues
- ✅ Plan production enhancements (RDS, ALB, Auto Scaling, monitoring)

## Next Steps

1. **Experiment**: Modify the Go API, add new endpoints, change the React UI
2. **Scale**: Add a second backend instance, configure load balancing
3. **Secure**: Implement authentication, add HTTPS, configure secrets management
4. **Optimize**: Analyze image sizes, implement caching, improve database queries
5. **Automate**: Set up full CI/CD pipeline with GitHub Actions + Terraform

---

## 📚 Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Reference](https://docs.docker.com/compose/)
- [Go Web Programming](https://go.dev/doc/code.html)
- [React Vite Guide](https://vite.dev/guide/)
- [AWS ECR User Guide](https://docs.aws.amazon.com/AmazonECR/latest/userguide/)
- [AWS EC2 Documentation](https://docs.aws.amazon.com/ec2/)

---

*This tutorial was created to help you learn Docker containerization and 3-tier application deployment. Happy coding!*