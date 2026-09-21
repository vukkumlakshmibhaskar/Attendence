FROM golang:1.23-bookworm AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/attendance-api ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/attendance-api /app/attendance-api
EXPOSE 8080
ENTRYPOINT ["/app/attendance-api"]
