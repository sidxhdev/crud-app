FROM golang:1.21-alpine

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./

RUN go build -o blog-backend .

USER appuser

EXPOSE 8080

CMD ["./blog-backend"]