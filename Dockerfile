FROM golang:1.19 AS build
WORKDIR /go/src
COPY internal/ambulance_wl ./internal/ambulance_wl
COPY main.go .
COPY go.sum .
COPY go.mod .

ENV CGO_ENABLED=0

RUN go build -o ambulance_wl .

FROM scratch AS runtime
ENV GIN_MODE=release
COPY --from=build /go/src/ambulance_wl ./
EXPOSE 8080/tcp
ENTRYPOINT ["./ambulance_wl"]
