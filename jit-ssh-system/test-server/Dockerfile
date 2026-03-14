# Use a base image specifically configured to run systemd correctly in Docker.
# This avoids many common pitfalls with cgroups and process management.
FROM jrei/systemd-ubuntu:22.04

# Avoid user interaction during apt installations
ENV DEBIAN_FRONTEND=noninteractive

# Install dependencies required for the agent and for SSH access
RUN apt-get update && apt-get install -y \
    openssh-server \
    curl \
    sudo \
    && rm -rf /var/lib/apt/lists/*

# Configure the SSH server to allow public key authentication and disallow passwords
RUN mkdir -p /var/run/sshd && \
    sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/PubkeyAuthentication no/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config

# Enable the ssh service so it starts on boot
RUN systemctl enable ssh

# Expose the standard SSH port
EXPOSE 22

# The base image's entrypoint will correctly start systemd as PID 1.
