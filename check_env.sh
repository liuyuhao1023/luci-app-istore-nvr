#!/bin/sh
echo "=== 1. OS & KERNEL ==="
cat /etc/openwrt_release 2>/dev/null
cat /etc/os-release 2>/dev/null
uname -a
if [ -f /etc/banner ]; then
  cat /etc/banner
fi

echo "=== 2. CPU & HARDWARE ==="
cat /proc/cpuinfo | grep -E "model name|cpu cores|processor" | head -n 16
echo -n "Total logical CPU cores: "
grep -c ^processor /proc/cpuinfo
uptime

echo "=== 3. MEMORY & SWAP ==="
free -m
swapon -s 2>/dev/null

echo "=== 4. DISK & MOUNTS ==="
df -hT
echo "--- lsblk ---"
lsblk -o NAME,SIZE,TYPE,FSTYPE,MOUNTPOINT,MODEL,TRAN 2>/dev/null || lsblk 2>/dev/null

echo "=== 5. STORAGE DEVICES ==="
fdisk -l 2>/dev/null | grep -E "Disk /dev/"

echo "=== 6. NETWORK INTERFACES ==="
ip -br addr 2>/dev/null || ifconfig -a
for i in /sys/class/net/*; do
  dev=$(basename "$i")
  speed=$(cat "$i/speed" 2>/dev/null)
  operstate=$(cat "$i/operstate" 2>/dev/null)
  echo "Interface: $dev | State: $operstate | Speed: ${speed:-unknown} Mbps"
done

echo "=== 7. PACKAGE MANAGER ==="
which opkg
opkg --version 2>/dev/null
echo "Installed packages count: $(opkg list-installed 2>/dev/null | wc -l)"

echo "=== 8. RUNTIME ENVIRONMENTS ==="
for cmd in docker dockerd containerd podman ffmpeg ffprobe gst-launch-1.0 sqlite3 go python3 node npm mount.cifs; do
  loc=$(which "$cmd" 2>/dev/null)
  if [ -n "$loc" ]; then
    echo "FOUND: $cmd at $loc"
  else
    echo "NOT FOUND: $cmd"
  fi
done

if which docker >/dev/null 2>&1; then
  docker info 2>/dev/null | grep -E "Server Version|Operating System|Storage Driver|Runtimes|Containers"
fi
if which ffmpeg >/dev/null 2>&1; then
  ffmpeg -version 2>/dev/null | head -n 2
  echo "FFmpeg H.264/H.265 decoders/encoders:"
  ffmpeg -encoders 2>/dev/null | grep -E "qsv|vaapi|h264|hevc"
fi
if which sqlite3 >/dev/null 2>&1; then
  sqlite3 --version
fi

echo "=== 8.1 CIFS / SMB MODULE CHECK ==="
lsmod | grep -E "cifs|smb"
modinfo cifs 2>/dev/null | head -n 5

echo "=== 9. WEB SERVICES & LISTENING PORTS ==="
netstat -tulnp 2>/dev/null || ss -tulnp 2>/dev/null

echo "=== 10. GPU & HARDWARE ACCELERATION ==="
lspci 2>/dev/null
ls -la /dev/dri 2>/dev/null
