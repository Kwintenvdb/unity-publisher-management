## Build

FROM golang:1.18-buster AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /upm-api-service

## Deploy

FROM gcr.io/distroless/base

WORKDIR /

COPY --from=build /upm-api-service /upm-api-service

EXPOSE 8080

USER nonroot:nonroot

CMD ["/upm-api-service"]
