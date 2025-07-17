# Crossplane provider image for Gitea
FROM alpine:3.19
RUN apk --no-cache add ca-certificates bash

ARG TARGETOS
ARG TARGETARCH

ADD _output/bin/${TARGETOS}_${TARGETARCH}/provider /usr/local/bin/provider

EXPOSE 8080
USER 65532
ENTRYPOINT ["provider"]