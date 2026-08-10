# Crossplane provider image for Gitea
FROM alpine:3.22.1

# OpenContainers Image Spec Labels
LABEL org.opencontainers.image.title="provider-gitea"
LABEL org.opencontainers.image.description="Crossplane provider for Gitea repository management"
LABEL org.opencontainers.image.vendor="rossigee"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/rossigee/provider-gitea"
LABEL org.opencontainers.image.url="https://github.com/rossigee/provider-gitea"
LABEL org.opencontainers.image.documentation="https://github.com/rossigee/provider-gitea/blob/master/README.md"

RUN apk --no-cache add ca-certificates bash

ARG TARGETOS
ARG TARGETARCH

ADD _output/bin/${TARGETOS}_${TARGETARCH}/provider /usr/local/bin/provider

EXPOSE 8080
USER 65532
ENTRYPOINT ["provider"]
