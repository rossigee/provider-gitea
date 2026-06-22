# Crossplane provider runtime image for Gitea.
#
# Built locally by scripts/build-xpkg.sh and embedded into the xpkg via
# `crossplane xpkg build --embed-runtime-image` — it is NOT pushed to the
# provider tag directly (that collision is the lesson-#5 bug the old release.yml
# shipped). distroless/static keeps it minimal; the Go binary is statically
# linked (CGO_ENABLED=0).
FROM gcr.io/distroless/static:nonroot

LABEL org.opencontainers.image.title="provider-gitea"
LABEL org.opencontainers.image.description="Crossplane v2 provider for Gitea"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/rossigee/provider-gitea"

ARG TARGETOS
ARG TARGETARCH

# The binary is already mode 0755 from `go build` and COPY preserves the mode,
# so we avoid `--chmod`, which would require BuildKit/buildx (lesson #13).
COPY bin/${TARGETOS}_${TARGETARCH}/provider /usr/local/bin/provider

# Copy the package (Provider meta + CRDs) for completeness/inspection.
COPY package/ /package/

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/provider"]
