FROM golang:1.18-alpine3.15 as builder
RUN apk add --update gcc musl-dev linux-headers git

# Build the simulator executable, from hive repo root.
# See context.txt for docker build context change.
# We use a go.work file to pull in other go modules of the hive repo.
ADD ./simulators/optimism/daisy-chain/hive.go.work /source/go.work
ADD ./optimism /source/optimism
ADD ./simulators/optimism/daisy-chain /source/daisy-chain

WORKDIR /source/daisy-chain
RUN go build -v .

# Build the simulator run container.
FROM alpine:latest

COPY --from=builder /source/daisy-chain/daisy-chain .
ENTRYPOINT ["./daisy-chain"]
