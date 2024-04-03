# Use the official Golang image as the base image
FROM golang:latest

# Args for ports
ARG HTTP_PORT
ARG TCP_PORT

# Set the working directory
WORKDIR /BalanceHub

# Copy the rest of the application source code
COPY . .

# Download and install dependencies
RUN go mod download

# Create app user
RUN useradd -m bhub

# Change ownership
RUN chown -R bhub:bhub .

# Set all permissions for the owner
RUN chmod 744 *

# Expose ports for http and tcp requests
EXPOSE $HTTP_PORT
EXPOSE $TCP_PORT

# Run the executable
ENTRYPOINT [ "./build.sh", "run"]
