#!/bin/bash

set -euo pipefail

domain=${CUSTOM_DOMAIN:-local-registry.io}

user="username"
passwd="DummyPasswd-$(date -u +%s)"

echo "Create certificate for ${domain}..."
cd /data && /bin/mkcert "${domain}"
ls -l /data/*.pem

echo "Fix /etc/hosts..."
if ! grep -q "${domain}" /etc/hosts; then
  echo "127.0.0.1 ${domain}" >> /etc/hosts
  echo "/etc/hosts updated"
fi
cat /etc/hosts

echo "Create credentials..."
htpasswd -Bbc /data/htpasswd "${user}" "${passwd}"
echo "Credentials user=${user} / passwd=${passwd}"
ls -l /data/htpasswd

echo "Launching registry..."
REGISTRY_HTTP_ADDR=0.0.0.0:443 \
  REGISTRY_HTTP_TLS_CERTIFICATE="/data/${domain}.pem" \
  REGISTRY_HTTP_TLS_KEY="/data/${domain}-key.pem" \
  REGISTRY_HTTP_RELATIVEURLS=true \
  REGISTRY_AUTH=htpasswd \
  REGISTRY_AUTH_HTPASSWD_PATH=/data/htpasswd \
  REGISTRY_AUTH_HTPASSWD_REALM="Registry Realm" \
  registry serve /data/registry-config.yml > /data/registry.log 2>&1 &

echo "Running $*"
LOCAL_REGISTRY_TEST=true \
  SSL_CERT_FILE="/data/${domain}.pem" \
	DOMAIN="${domain}" \
	USER="${user}" PASSWD="${passwd}" "$@"
