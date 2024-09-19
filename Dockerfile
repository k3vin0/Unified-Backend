# Use the official Go image as the base
FROM golang:latest

# Set the working directory inside the container
WORKDIR /app

# Install git to clone the repository
RUN apt-get update && apt-get install -y git

# Clone the Air repository and build from source
RUN git clone https://github.com/cosmtrek/air.git /tmp/air && \
    cd /tmp/air && \
    go build -o /go/bin/air

# Copy the Go module files to download dependencies first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Ensure correct permissions for the .env file
RUN chmod 644 .env

# Expose the port your application will run on
EXPOSE 42069

# Command to run the application using Air
CMD ["air", "-c", ".air.toml"]