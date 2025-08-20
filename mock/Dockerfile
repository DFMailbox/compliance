# syntax=docker/dockerfile:1

##
## Build the application from source
##

FROM golang:1.24 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /mock-server

##
## Deploy the application binary into a lean image
##

FROM gcr.io/distroless/base-debian12 AS build-release-stage

WORKDIR /

COPY --from=build-stage /mock-server /mock-server

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/mock-server"]
