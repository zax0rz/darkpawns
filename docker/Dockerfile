# Dark Pawns portable image, built from the signed release archives.
#
# Build context must contain this run's dist/ (the signed archives plus
# checksums.txt GoReleaser generated over them). The bundle stage verifies the
# per-arch checksum before extraction; at run time the launcher's own strict
# publisher gate re-verifies the bundle signature, so an image cannot smuggle
# bytes the release did not sign.
#
# The entrypoint initializes /data on first boot (the launcher's run command
# expects an existing instance), fixes the volume's ownership, then drops to
# an unprivileged user. The launcher binds HTTP to loopback inside the
# container, so publish ports with --network host until an instance-level
# HTTP bind exists.
FROM debian:stable-slim AS bundle
ARG TARGETARCH
WORKDIR /tmp
COPY dist/checksums.txt .
COPY dist/darkpawns_*_linux_*.tar.gz ./
RUN cp darkpawns_*_linux_${TARGETARCH}.tar.gz bundle.tar.gz \
 && grep "linux_${TARGETARCH}\.tar\.gz$" checksums.txt | awk '{print $1"  bundle.tar.gz"}' > bundle.sha \
 && sha256sum -c --strict bundle.sha \
 && mkdir /app \
 && tar -xzf bundle.tar.gz --strip-components=1 -C /app \
 && rm -f bundle.tar.gz bundle.sha checksums.txt darkpawns_*_linux_*.tar.gz

FROM debian:stable-slim
RUN useradd --system --uid 65532 --home /app darkpawns
COPY --from=bundle --chown=darkpawns:darkpawns /app /app
COPY docker/entrypoint.sh /entrypoint.sh
VOLUME /data
USER root
EXPOSE 4350 7777
ENTRYPOINT ["/entrypoint.sh"]
