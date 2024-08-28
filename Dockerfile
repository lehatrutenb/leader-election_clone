FROM golang:1.22 AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

ADD ./ /app

RUN CGO_ENABLED=0 GOOS=linux go build -o build/election cmd/election/main.go

FROM alpine:latest

COPY --from=build /app/build/* /opt/
EXPOSE 8080

ENTRYPOINT [ "/opt/election" ]
CMD [ "run" ]

