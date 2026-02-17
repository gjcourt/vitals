FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /biometrics ./cmd/biometrics

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /biometrics /usr/local/bin/biometrics
COPY --from=builder /src/web /web
ENV WEB_DIR=/web
EXPOSE 8080
USER 65534:65534
ENTRYPOINT ["biometrics"]
