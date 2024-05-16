FROM golang:1.22 as builder

RUN mkdir /app
ADD . /app/
WORKDIR /app

RUN CGO_ENABLED=0 go build -installsuffix 'static' -o /app/main \
 && go mod tidy \
 && go clean --cache

FROM scratch

COPY --from=builder /app/main /sidecar-cleaner

CMD ["/sidecar-cleaner"]