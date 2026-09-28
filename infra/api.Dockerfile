FROM golang:1.22-alpine AS build
WORKDIR /src
COPY apps/api/go.mod ./
COPY apps/api/ ./
RUN CGO_ENABLED=0 go build -o /out/opspilot ./cmd/opspilot

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/opspilot /usr/local/bin/opspilot
EXPOSE 8094
USER nobody
ENTRYPOINT ["opspilot"]
