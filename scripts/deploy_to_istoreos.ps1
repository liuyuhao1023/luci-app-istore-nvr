# iStore NVR 一键编译与软路由部署脚本
param(
    [string]$RouterIP = "192.168.1.15"
)

Write-Host ">>> 1. 正在编译 Vue 3 前端静态资源..." -ForegroundColor Cyan
Set-Location "d:\监控开发\frontend"
cmd.exe /c "npm run build-only"

Write-Host ">>> 2. 正在交叉编译 Linux x86_64 纯静态后端二进制..." -ForegroundColor Cyan
Set-Location "d:\监控开发\backend"
$env:Path = "C:\Program Files\Go\bin;" + $env:Path
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -ldflags="-s -w" -o istore-nvr ./cmd/server

Write-Host ">>> 3. 正在同步程序与资源到软路由 /mnt/sata1-4/istore-nvr/ ..." -ForegroundColor Cyan
ssh -o StrictHostKeyChecking=no -i "$HOME\.ssh\id_ed25519" root@$RouterIP "/etc/init.d/istore-nvr stop 2>/dev/null; mkdir -p /mnt/sata1-4/istore-nvr/dist /mnt/sata1-4/istore-nvr/data"
scp -o StrictHostKeyChecking=no -i "$HOME\.ssh\id_ed25519" "d:\监控开发\backend\istore-nvr" root@${RouterIP}:/mnt/sata1-4/istore-nvr/istore-nvr
scp -r -o StrictHostKeyChecking=no -i "$HOME\.ssh\id_ed25519" "d:\监控开发\backend\dist\*" root@${RouterIP}:/mnt/sata1-4/istore-nvr/dist/

Write-Host ">>> 4. 正在赋予权限并启动服务..." -ForegroundColor Cyan
ssh -o StrictHostKeyChecking=no -i "$HOME\.ssh\id_ed25519" root@$RouterIP "chmod +x /mnt/sata1-4/istore-nvr/istore-nvr && /etc/init.d/istore-nvr restart"

Write-Host ">>> 部署完成！请在浏览器中打开: http://${RouterIP}:8080/" -ForegroundColor Green
