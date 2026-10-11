# syntax=docker/dockerfile:1
ARG GO_IMAGE=golang:1.27.2-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61
FROM ${GO_IMAGE} AS dependencies
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify

FROM dependencies AS test
COPY . .
CMD ["make", "verify"]

FROM dependencies AS build
ARG VERSION=dev
ARG REVISION=unknown
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -X main.version=${VERSION} -X main.revision=${REVISION}" -o /out/portway ./cmd/portway
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/events ./cmd/events

FROM scratch AS events-runtime
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/events /events
USER 65532:65532
EXPOSE 8081
HEALTHCHECK --interval=15s --timeout=5s --start-period=30s CMD ["/events", "probe"]
ENTRYPOINT ["/events"]
CMD ["relay"]

FROM scratch AS runtime
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/portway /portway
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=4s --start-period=5s CMD ["/portway", "probe"]
ENTRYPOINT ["/portway"]
CMD ["api"]
