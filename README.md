# OpenWrt NVR - 网络视频监控管理系统 (NVR 摄像头管理)

<p align="center">
  <img src="https://img.shields.io/badge/Platform-OpenWrt%20%7C%20iStoreOS%20%7C%20Linux-blue.svg" alt="Platform">
  <img src="https://img.shields.io/badge/Architecture-x86__64%20%7C%20aarch64-green.svg" alt="Architecture">
  <img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8.svg" alt="Go">
  <img src="https://img.shields.io/badge/Vue-3.4+-4FC08D.svg" alt="Vue">
</p>

专为运行在 **OpenWrt / Linux 软路由（广泛支持原生 OpenWrt、iStoreOS、ImmortalWrt 及通用 Linux 发行版）** 上打造的高性能、免插件、低功耗网络视频监控录像系统（NVR）。

让软路由低功耗替代传统 NVR 硬件录像机，实现摄像头自动发现、跨网段接入、WebRTC 极低延迟多画面预览、原始码流直通切片录像与局域网 NAS (SMB/CIFS) 网络存储管理。

---

## 🌟 核心特性

- 🚀 **专为软路由优化**：纯静态 Go 后端 + 嵌入式 Vue 3 前端，常驻内存仅约 16MB，CPU 空载占用接近 0%，绝不影响路由器网络转发基础功能。
- 🌐 **跨网段摄像头深度适配**：支持多网段并发单播+组播 WS-Discovery 探测，支持指定软路由物理出口网卡（如 eth0/eth1），解决安防专网 VLAN 或跨网段接入难题。
- 🎥 **现代浏览器全平台免插件**：彻底淘汰老旧 IE 和 ActiveX 插件噩梦！基于 **WebRTC (WHEP) + HLS** 零转码直通播放，Chrome、Edge、Safari、Firefox 及手机平板原生秒开（延迟 < 0.3 秒）。
- ⏹️ **录像与预览解耦（独立录像开关）**：提供系统全局录像总开关与单摄像头独立录像开关，关闭录像不影响实时预览；真实 REC 闪烁徽章联动显示。
- 💾 **SMB/CIFS 网络存储 (NAS) 无缝集成**：图形化挂载局域网群晖、威联通或 Windows 共享，具备**断网脱机熔断保护**，严禁将录像文件错误写入软路由系统盘或 `/overlay` 分区。
- 📱 **1 ~ 64 宫格自适应排布**：单画面高清主码流、多画面流畅子码流动态策略，避免压垮软路由带宽与客户端解码能力。
- 🧩 **原生 LuCI 插件支持**：提供标准 `luci-app-istore-nvr` 插件，无缝嵌入 iStoreOS / OpenWrt 系统后台与应用商店。

---

## 🏗️ 系统架构图

```
+-----------------------------------------------------------------------------------+
|                        Web 管理中心 (PC / 平板 / 手机浏览器)                       |
|           Vue 3 + Vite + Element Plus + WebRTC (WHEP) / HLS 极低延迟播放器         |
+------------------------------------------+----------------------------------------+
                                           | HTTP REST API / WebRTC WHEP
+------------------------------------------v----------------------------------------+
|                      OpenWrt NVR 核心服务 (Linux x86_64 / arm64)                   |
|  +-----------------------------------------------------------------------------+  |
|  | 设备适配层: 海康威视专属适配 (ISAPI / Digest鉴权 / RTSP TCP 码流自适应)     |  |
|  +-----------------------------------------------------------------------------+  |
|  | 跨网段探测引擎: 多网段并行单播 Probe + 组播 WS-Discovery + 端口探针         |  |
|  +-----------------------------------------------------------------------------+  |
|  | 流媒体网关: 深度集成 MediaMTX (流复用分发，RTSP -> WebRTC WHEP / HLS)       |  |
|  +-----------------------------------------------------------------------------+  |
|  | 独立录像引擎: 零重编码直通切片 (fMP4/TS)，按摄像头独立任务调度与防误写熔断  |  |
|  +-----------------------------------------------------------------------------+  |
|  | 存储与索引: SQLite WAL 模式 (存储于本地 SATA 数据盘或 SMB NAS 挂载点)       |  |
+-----------------------------------------------------------------------------------+
```

---

## 📦 安装与部署指南

### 方式一：通过 IPK 插件安装（推荐）

1. 在 GitHub Releases 页面下载最新的 `luci-app-istore-nvr_*.ipk` 与对应的二进制包；
2. 上传至软路由，执行安装：
   ```sh
   opkg update
   opkg install luci-app-istore-nvr_1.0.0-1_all.ipk
   ```
3. 登录 iStoreOS / OpenWrt 路由器后台，在导航栏中找到 **【服务】->【iStore NVR 监控】** 即可启用与配置。

### 方式二：手动独立运行

1. 下载编译好的静态二进制程序 `istore-nvr` 与前端 `dist` 目录；
2. 放置到本地数据盘（例如 `/mnt/sata1-4/istore-nvr/`）：
   ```sh
   mkdir -p /mnt/sata1-4/istore-nvr/data
   chmod +x /mnt/sata1-4/istore-nvr/istore-nvr
   /mnt/sata1-4/istore-nvr/istore-nvr -port 8080 -db /mnt/sata1-4/istore-nvr/data/nvr.db
   ```
3. 浏览器访问：`http://<软路由IP>:8080/`

---

## ⚙️ 摄像头与存储配置示例

### 1. 海康威视摄像头常见接入地址
- **主码流（高清）**：`rtsp://admin:密码@摄像头IP:554/Streaming/Channels/101`
- **子码流（流畅）**：`rtsp://admin:密码@摄像头IP:554/Streaming/Channels/102`
- 系统默认采用 **RTSP over TCP** 传输，并在添加摄像头时提供自动探测与连通性测试。

### 2. SMB 网络存储 (NAS) 配置
- 在管理后台进入 **【网络与本地存储 (SMB)】**；
- 填入 NAS 服务器 IP（如 `192.168.1.200`）、共享目录名、专享访问账号与密码；
- 系统自动挂载并验证读写权限；脱机时自动熔断暂停录像，保护软路由本地存储空间。

---

## 🛠️ 本地编译与构建

### 1. 前端构建
```sh
cd frontend
npm install
npm run build-only
```

### 2. 交叉编译 Linux 纯静态二进制
```sh
cd backend
# 编译 x86_64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o istore-nvr ./cmd/server

# 编译 arm64 (适用于友善 R4S/R5S/RK3588 等设备)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o istore-nvr-arm64 ./cmd/server
```

### 3. 一键打包 OpenWrt IPK
```sh
sh scripts/build_ipk.sh
# 产物输出在 bin/luci-app-istore-nvr_1.0.0-1_all.ipk
```

---

## 🛡️ 安全规范

1. **零密码明文**：所有摄像头密码与 SMB 密码均采用 AES-GCM 高强度密文存储，Web API 绝不返回密码明文；
2. **闪存保护红线**：严禁将录像文件和临时日志写入软路由系统分区（`/overlay`）以防止设备死机变砖；录像必须保存在数据盘（`/mnt/sata*`、外接移动硬盘或 SMB NAS）中。

---

## 📄 开源许可证

本项目采用 [MIT License](LICENSE) 许可协议开源。
