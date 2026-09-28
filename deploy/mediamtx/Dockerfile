FROM golang:1.26.6-alpine AS build
RUN apk add --no-cache git
WORKDIR /src
# Pin the version currently used by the cloud deployment; never patch latest.
RUN git clone --depth 1 --branch v1.19.3 https://github.com/bluenviron/mediamtx.git . \
    && test "$(git rev-parse HEAD)" = "5ccc80ad7b1609330a4c397c51a7e701f963d74f"
COPY taiwan-recording.patch /tmp/taiwan-recording.patch
RUN git apply --check /tmp/taiwan-recording.patch && git apply /tmp/taiwan-recording.patch
COPY recording_boundary.go.src ./internal/recorder/recording_boundary.go
COPY recording_boundary_test.go.src ./internal/recorder/recording_boundary_test.go
RUN go test ./internal/recorder ./internal/playback
RUN go generate ./internal/core ./internal/servers/hls
RUN CGO_ENABLED=0 go build -trimpath -o /out/mediamtx .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=build /out/mediamtx /mediamtx
ENTRYPOINT ["/mediamtx"]
