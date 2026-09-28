FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /demo ./cmd/demo

FROM alpine:3.20
RUN adduser -D -u 65532 app
USER 65532
COPY --from=build /demo /demo
EXPOSE 8080
ENTRYPOINT ["/demo"]
