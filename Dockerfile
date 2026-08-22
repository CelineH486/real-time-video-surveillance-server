FROM ghcr.io/cirruslabs/flutter:stable AS flutter-build
WORKDIR /src/apps/mobile
COPY apps/mobile/pubspec.yaml apps/mobile/pubspec.lock ./
RUN flutter pub get
COPY apps/mobile/ ./
RUN flutter build web --release --base-href /web/ --no-web-resources-cdn --no-wasm-dry-run

FROM golang:1.26.6-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=flutter-build /src/apps/mobile/build/web/ ./web/
RUN CGO_ENABLED=0 go build -trimpath -o /out/server .

FROM alpine:3.22
RUN addgroup -S app && adduser -S app -G app
USER app
COPY --from=build /out/server /server
EXPOSE 8080/tcp 5000/udp
ENTRYPOINT ["/server"]
