FROM golang:1.13 as builder
LABEL maintainer="Rikanishu <adantess@gmail.com>"
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .
FROM alpine:latest 
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/app .
ADD ./config.yaml ./config.yaml
EXPOSE 12950 12951
CMD ["./app"]
