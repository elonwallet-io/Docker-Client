FROM golang:1.20.2 as build-env

ENV CGO_ENABLED=0

COPY . /app/
WORKDIR /app/

RUN go build -o relay -ldflags="-extldflags=-static" .

FROM gcr.io/distroless/static

COPY --from=build-env --chown=nonroot:nonroot \
    /app/relay \
    .

USER nonroot:nonroot

EXPOSE 8080

CMD ["./relay"]