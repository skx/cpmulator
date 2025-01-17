#
# Trivial Dockerfile for cpmulator, which contains our sample binaries
#
# Build it like so:
#
#   docker build -t cpmulator .
#
# Launch it like so to run "all the things":
#
#   docker run -t -i cpmulator:latest
#
# Or like this to be more specific:
#
#   docker run -t -i cpm:latest -input stty -cd /app/G /app/G/ZORK1.COM
#
#

# STEP1 - Build-image
###########################################################################
FROM golang:alpine AS builder

ARG VERSION

LABEL org.opencontainers.image.source="https://github.com/skx/cpmulator/"
LABEL org.opencontainers.image.description="CP/M Emulator written in Golang"

# Create a working-directory
WORKDIR $GOPATH/src/github.com/skx/cpmulator/

# Copy the source to it
COPY . .

# Build the binary - ensuring we pass the build-argument
RUN go build -ldflags "-X main.version=$VERSION" -o /go/bin/cpmulator

# STEP2 - Deploy-image
###########################################################################
FROM alpine

# Copy the binary.
COPY --from=builder /go/bin/cpmulator /usr/local/bin/

# Create a group and user
RUN addgroup app && adduser -D -G app -h /apps app

# Clone the binaries to test with
RUN apk add git && git clone https://github.com/skx/cpm-dist.git /app && chown -R app:app /app

# Tell docker that all future commands should run as the app user
USER app

# Set working directory
WORKDIR /app

# Set entrypoint
ENTRYPOINT [ "/usr/local/bin/cpmulator" ]

# And ensure we use the subdirectory-support.
CMD [ "-directories" ]
