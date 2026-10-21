# Build Image
FROM debian:trixie-20260824-slim@sha256:d7e12182ce18b85b93007c1dedf31f2d29e01ccf3182cc4017c709b6259bc132 AS toolchain
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
FROM golang:1.27.2-trixie@sha256:47667934874913a449b96faa7ef4777fa285d9ec29126ee80ed61c45b7cccf0e AS tool
WORKDIR /tool
COPY tool/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags=-buildid= -o /system-tool .

# Sources
FROM scratch AS sources
ADD --chmod=644 --checksum=sha256:f1c8157570bdc452163db2ff109978246cd25b4e93d3c4c9fefaeee3c4d6982d \
    https://buildroot.org/downloads/buildroot-2026.05.2.tar.gz /buildroot-2026.05.2.tar.gz
ADD --chmod=644 --checksum=sha256:5c18c6729e1df68a411a6e01f81f8b52e9defb453d17b53f04b9330fc8e9a53e \
    https://repo.hex.pm/tarballs/nerves_system_br-1.34.3.tar /nerves_system_br-1.34.3.tar

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
