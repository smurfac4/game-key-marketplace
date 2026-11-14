#!/bin/sh
set -eu
export HTTP_ADDR=":${PORT:-8080}"
export AUTH_ALLOWED_ORIGINS="${AUTH_ALLOWED_ORIGINS:-${RENDER_EXTERNAL_URL:?Render must supply its public HTTPS URL}}"
export AUTH_WEB_URL="${AUTH_WEB_URL:-$AUTH_ALLOWED_ORIGINS}"
migrate up
seed -if-empty
exec server
