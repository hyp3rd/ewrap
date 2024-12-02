# syntax=docker/dockerfile:1

# Dockerfile reference guide at
# https://docs.docker.com/go/dockerfile-reference/

################################################################################
# Create a stage for building the application.
ARG GO_VERSION=1.23.2
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
WORKDIR /src

# Download dependencies as a separate step to take advantage of Docker's caching.
# Leverage a cache mount to /go/pkg/mod/ to speed up subsequent builds.
# Leverage bind mounts to go.sum and go.mod to avoid having to copy them into
# the container.
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

# This is the architecture you're building for, which is passed in by the builder.
# Placing it here allows the previous steps to be cached across architectures.
ARG TARGETARCH=amd64

ENV VERSION=""

COPY ./configs /src/configs
COPY ./docs /src/docs

# Build the application.
# Leverage a cache mount to /go/pkg/mod/ to speed up subsequent builds.
# Leverage a bind mount to the current directory to avoid having to copy the
# source code into the container.
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOARCH=$TARGETARCH go build \
    -ldflags="all=-s -w -X main.Version=$VERSION -X main.BuildTime=$(date +%FT%T%z)" \
    -tags=!healthcheck -trimpath -o /bin/server ./cmd/app/ && \
    # Build the healthcheck binary.
    CGO_ENABLED=0 GOARCH=$TARGETARCH go build \
    -ldflags="all=-s -w" \
    -trimpath -tags=healthcheck -o /bin/healthcheck ./cmd/healthcheck/main.go

# Create a non-privileged user that the app will run under.
# See https://docs.docker.com/go/dockerfile-user-best-practices/
RUN addgroup -S appgroup && \
    adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    appuser -G appgroup  && \
    # prepare the bin folder to receive the build's binaries
    mkdir -p /bin

################################################################################
# Create a new stage for running the application that contains the minimal
# runtime dependencies for the application. This often uses a different base
# image from the build stage where the necessary files are copied from the build
# stage.
FROM scratch AS app

# Copy the non-root user and group files from the builder stage
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group

# Copy the app executable from the "build" stage.
COPY --from=build /bin/server /bin/
# Copy the healthcheck executable from the "build" stage.
COPY --from=build /bin/healthcheck /bin/
# Copy the configuration files from the "build" stage.
COPY --from=build /src/configs/ /configs/

# Set the non-root user
USER appuser:appgroup

# Expose the port that the application listens on.
EXPOSE 8000

HEALTHCHECK --interval=60s --timeout=10s --start-period=10s --retries=3 CMD ["/bin/healthcheck"]

# What the container should run when it is started.
ENTRYPOINT [ "/bin/server" ]
