# Build Image
FROM debian:trixie-20261104-slim@sha256:a72b4433de1cf7d6bae0c4cee74d0547f58601a643b3375f070c51600d63b6b2 AS toolchain
RUN sed -i -e '/^URIs:/d' -e 's|^# \(http://snapshot\.debian\.org/.*\)$|URIs: \1|' \
    /etc/apt/sources.list.d/debian.sources \
    && grep -q '^URIs: http://snapshot.debian.org/' /etc/apt/sources.list.d/debian.sources \
    && apt-get -o Acquire::Check-Valid-Until=false update \
    && apt-get install -y --no-install-recommends \
    bc build-essential bzip2 ca-certificates cpio file git patch perl python3 rsync unzip wget xz-utils \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --uid 1000 --create-home builder \
    && install -d -o builder -g builder /work
ENV LANG=C.UTF-8

# Build Tool
FROM golang:1.27.4-trixie@sha256:f18c3d73a184fb0582828510722e723b63a93f7188d11dd223d9be6f6880e65d AS tool
WORKDIR /tool
COPY tool/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags=-buildid= -o /system-tool .

# Sources
FROM scratch AS sources
ADD --chmod=644 --checksum=sha256:38a6c152490096018720a9ade379ee1e571c9bf5f93a3657a166566945a98b0d \
    https://buildroot.org/downloads/buildroot-2026.08.1.tar.gz /buildroot-2026.08.1.tar.gz
ADD --chmod=644 --checksum=sha256:6b8bff33262bb031aee6ea79c1c2686e40c833c3d2345ef1a20c6563375f732d \
    https://repo.hex.pm/tarballs/nerves_system_br-1.35.3.tar /nerves_system_br-1.35.3.tar

# Build the System
FROM toolchain AS system
ARG TARGET
COPY --from=tool /system-tool /usr/local/bin/
COPY VERSION /src/
COPY config/ /src/config/
COPY boot/ /src/boot/
COPY buildroot/ /src/buildroot/
COPY fwup/ /src/fwup/
COPY patches/ /src/patches/
COPY rootfs/ /src/rootfs/
USER builder
WORKDIR /work

RUN --mount=type=bind,from=sources,target=/sources \
    system-tool configure -target "${TARGET}"

RUN --mount=type=cache,id=buildroot-downloads,target=/cache,uid=1000,gid=1000,sharing=locked \
    system-tool download

ARG SOURCE_DATE_EPOCH=0
RUN --network=none system-tool build -target "${TARGET}"

ARG COMMIT=unknown
RUN --network=none system-tool package -target "${TARGET}" -commit "${COMMIT}"

FROM scratch AS release
COPY --from=system /work/release/ /
