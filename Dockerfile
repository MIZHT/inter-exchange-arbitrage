FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/market-data ./cmd/bot \
	&& CGO_ENABLED=0 go build -trimpath -o /out/snapshot-service ./cmd/snapshot_receiver

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/market-data ./market-data
COPY --from=build /out/snapshot-service ./snapshot-service
RUN mkdir -p /app/data /app/out/logs

