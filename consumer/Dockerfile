# Use an official Golang runtime as the parent image
FROM golang:1.21

# Set the working directory in the container
WORKDIR /app

# Copy the local package files to the container's workspace.
ADD . /app

# Install the application dependencies and binaries.
# RUN go mod init simple-webserver
RUN go get -d -v ./...
RUN go install -v ./...

# Make port 8080 available to the world outside this container
EXPOSE 8080

# Run the executable
CMD ["simple-webserver"]
