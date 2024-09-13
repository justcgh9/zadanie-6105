FROM golang:1.23

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /docker-gs-ping ./cmd/tender-system


EXPOSE 8080

# Run
CMD ["/docker-gs-ping"]