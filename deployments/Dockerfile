FROM golang:1.23-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o /bin/server ./cmd/server

FROM alpine:3.18
COPY --from=build /bin/server /bin/server
EXPOSE 8080
ENTRYPOINT ["/bin/server"]
