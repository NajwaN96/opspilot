FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY main.go .
RUN go build -o /service main.go
FROM alpine:3.22
RUN adduser -D -u 65532 app
COPY --from=build /service /service
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/service"]
