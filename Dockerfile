# Use the official Debian image as the base for building
FROM golang:latest AS build

# Args
ENV WEBUI_BASE_URL=http://localhost:8080
ENV WG_GATEWAY_IP=10.2.0.1
ENV WEBUI_USERNAME=
ENV WEBUI_PASSWORD=

# Set the working directory
WORKDIR /app

# Copy the Go source code
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o update-natpmp main.go

# Use the official Debian image as the base for running
FROM debian:latest

# Install required dependencies
RUN apt update && apt install -y \
        natpmpc \
    && rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /app

# Copy the compiled Go app, entry-point script, and make them executable
COPY --from=build /app/update-natpmp .
COPY ./entrypoint.sh .
RUN chmod +x ./update-natpmp ./entrypoint.sh

# Set the entry point
ENTRYPOINT ["./entrypoint.sh"]
