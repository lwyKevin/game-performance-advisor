# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/game-performance-advisor ./cmd/game-performance-advisor

FROM scratch
COPY --from=build /out/game-performance-advisor /game-performance-advisor
COPY samples /samples
WORKDIR /samples
ENTRYPOINT ["/game-performance-advisor"]
CMD ["analyze", "/samples/smooth_60fps.csv", "--target-fps", "60"]
