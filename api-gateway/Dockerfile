## Build

FROM golang:1.18-buster AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /upm-ag

## Deploy

FROM gcr.io/distroless/base

WORKDIR /

COPY --from=build /upm-ag /upm-ag

EXPOSE 8080

USER nonroot:nonroot

CMD ["/upm-ag"]
