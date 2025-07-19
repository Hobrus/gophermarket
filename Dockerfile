# Build stage
FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o gophermart ./cmd/gophermart

# Run stage
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/gophermart /app/gophermart
EXPOSE 8080
ENTRYPOINT ["/app/gophermart"]
