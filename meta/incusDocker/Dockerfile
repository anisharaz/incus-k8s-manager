# Use the official Debian Bookworm base image
FROM debian:bookworm-slim

# We make a fake systemctl so that incus doesn't error out without systemd
RUN echo "#!/bin/bash" > /sbin/systemctl && \
    echo "exit 0" >> /sbin/systemctl && \
    chmod +x /sbin/systemctl && \
    echo "deb http://deb.debian.org/debian bookworm contrib" >> /etc/apt/sources.list && \
    # Install curl so we can install the keyring.
    apt-get update && \
    apt-get install --no-install-recommends -y curl ca-certificates && \
    mkdir -p /etc/apt/keyrings/ && \
    curl -fsSL https://pkgs.zabbly.com/key.asc -o /etc/apt/keyrings/zabbly.asc && \
    echo "deb [signed-by=/etc/apt/keyrings/zabbly.asc] https://pkgs.zabbly.com/incus/stable $(. /etc/os-release && echo ${VERSION_CODENAME}) main" > /etc/apt/sources.list.d/zabbly-incus-stable.list && \
    # Install incus and dependencies
    # Added 'passwd' to ensure groupmod is available for GID mapping
    apt-get update && \
    apt-get install --no-install-recommends -y fuse nftables ebtables arptables iproute2 thin-provisioning-tools openvswitch-switch btrfs-progs lvm2 udev iptables kmod passwd socat && \
    apt-get install --no-install-recommends --no-install-suggests -y zfsutils-linux && \
    apt-get install --no-install-recommends -y incus incus-ui-canonical && \
    apt-get remove -y curl && \
    apt autoremove -y && \
    apt-get clean

# Copy the entrypoint script and preseed config into the image
COPY entrypoint.sh /entrypoint.sh
COPY ./incusStuff/incus_admin_config.yaml /incus_admin_config.yaml
RUN chmod +x /entrypoint.sh

# Bake the prebuilt k8s VM image into the image so entrypoint.sh can import
# it into Incus during the one-time init step.
# These files come from the "vmimage" named build context — a temp dir that
# build_docker_image.sh creates in /tmp and cleans up after the build.
COPY --from=vmimage /incus.tar.xz /incus-images/incus.tar.xz
COPY --from=vmimage /disk.qcow2 /incus-images/disk.qcow2

# Healthcheck: verify incusd responds on the local API socket
HEALTHCHECK --interval=10s --timeout=5s --start-period=120s --retries=12 \
    CMD incus query /1.0 >/dev/null 2>&1 || exit 1

# Run the incusd binary via entrypoint.sh
CMD ["/entrypoint.sh"]