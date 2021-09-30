FROM --platform=$BUILDPLATFORM quay.io/containers/skopeo:latest
COPY operator-manifest-tools /usr/bin/operator-manifest-tools
ENTRYPOINT ["/usr/bin/operator-manifest-tools"]
