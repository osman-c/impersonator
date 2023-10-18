# Use the official Golang image as the base image
FROM golang:1.18

WORKDIR /bot

# Copy your Go application source code into the container
COPY . .

# Build the Go application
RUN go build -o impersonator

# Specify the command to run when the container starts
CMD ["./impersonator"]











