ARG GO_VERSION=1.12

FROM golang:${GO_VERSION}-alpine AS builder
RUN apk add --no-cache ca-certificates git
WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./ ./
RUN go build -o /app .


FROM alpine AS final
COPY --from=builder /app /app
EXPOSE 8080
ENTRYPOINT ["/app"]