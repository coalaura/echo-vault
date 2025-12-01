#!/bin/bash

set -euo pipefail

echo "Linking sysusers config..."

mkdir -p /etc/sysusers.d

if [ ! -f /etc/sysusers.d/echo_vault.conf ]; then
    ln -s "/var/images.shrt.day/conf/echo_vault.conf" /etc/sysusers.d/echo_vault.conf
fi

echo "Creating user..."
systemd-sysusers

echo "Linking unit..."
rm /etc/systemd/system/echo_vault.service

systemctl link "/var/images.shrt.day/conf/echo_vault.service"

echo "Reloading daemon..."
systemctl daemon-reload
systemctl enable echo_vault

echo "Fixing initial permissions..."
chown -R echo_vault:echo_vault "/var/images.shrt.day"

find "/var/images.shrt.day" -type d -exec chmod 755 {} +
find "/var/images.shrt.day" -type f -exec chmod 644 {} +

chmod +x "/var/images.shrt.day/echo_vault"

echo "Setup complete, starting service..."

service echo_vault start

echo "Done."
