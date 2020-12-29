# First, build app
FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download 
COPY . .
RUN go build -o rpi-webcam . 

# For the app
EXPOSE 4443 
# For Let's Encrypt
EXPOSE 4444

ENTRYPOINT ["/app/rpi-webcam"]
