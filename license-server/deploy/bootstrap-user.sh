#!/usr/bin/env bash
# One-time, as root: create the dedicated "scadabench" deploy login so root is not used for
# day-to-day deploys. It copies the ScadaBench deploy key from root's authorized_keys only.
#   ssh root@host 'bash -s' < license-server/deploy/bootstrap-user.sh
set -euo pipefail
USER_NAME=scadabench
KEY_TAG="scadabench-deploy"
[[ $EUID -eq 0 ]] || { echo "run as root"; exit 1; }
id "$USER_NAME" >/dev/null 2>&1 || useradd --create-home --shell /bin/bash "$USER_NAME"
install -d -m 0700 -o "$USER_NAME" -g "$USER_NAME" "/home/$USER_NAME/.ssh"
grep "$KEY_TAG" /root/.ssh/authorized_keys > "/home/$USER_NAME/.ssh/authorized_keys"
chown "$USER_NAME:$USER_NAME" "/home/$USER_NAME/.ssh/authorized_keys"
chmod 0600 "/home/$USER_NAME/.ssh/authorized_keys"
echo "$USER_NAME ALL=(root) NOPASSWD:SETENV: /usr/bin/bash /tmp/scadabench-ls/deploy/setup.sh*, NOPASSWD: /usr/bin/install -d -m 0700 /etc/scadabench, /usr/bin/tee /etc/scadabench/tunnel.env, /usr/bin/chmod 0600 /etc/scadabench/tunnel.env, /usr/local/bin/license-server-admin *, /usr/bin/systemctl * license-server, /usr/bin/systemctl * scadabench-tunnel, /usr/bin/journalctl *" > /etc/sudoers.d/scadabench
chmod 0440 /etc/sudoers.d/scadabench
visudo -cf /etc/sudoers.d/scadabench
echo "User $USER_NAME ready: ssh $USER_NAME@$(hostname -I | awk '{print $1}')"
