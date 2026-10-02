#!/bin/sh
# Auto-configuration hook for nvr-manager in iStoreOS
set -e

[ -n "$ISTORE_CONF_DIR" ] || exit 0

data_dir="$ISTORE_CONF_DIR/nvr-manager/data"
record_dir="$ISTORE_CONF_DIR/nvr-manager/recordings"

mkdir -p "$data_dir" "$record_dir"

enabled='1'
if [ -n "$ISTORE_DONT_START" ] && [ "$ISTORE_DONT_START" = "1" ]; then
	enabled='0'
fi

uci batch <<EOF
set nvr-manager.config.data_dir='$data_dir'
set nvr-manager.config.record_dir='$record_dir'
set nvr-manager.config.enabled='$enabled'
commit nvr-manager
EOF

if [ "$enabled" = "1" ]; then
	/etc/init.d/nvr-manager restart 2>/dev/null || true
fi

exit 0
