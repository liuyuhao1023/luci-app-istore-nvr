local m, s, o
local uci = require "luci.model.uci".cursor()
local sys = require "luci.sys"

local is_running = (sys.call("pidof istore-nvr >/dev/null") == 0)
local port = uci:get("istore-nvr", "config", "port") or "8080"
local host = luci.http.getenv("SERVER_NAME") or "192.168.1.15"

m = Map("istore-nvr", translate("iStore NVR 网络视频监控管理系统"))
m.description = translate("专为软路由定制的高性能轻量级 NVR 系统，支持海康威视等摄像头跨网段直通录像、多画面低延迟实时预览与 SMB 网络存储。")

s = m:section(NamedSection, "config", "istore-nvr", translate("服务运行状态与操作"))
s.anonymous = true

-- 状态显示
o = s:option(DummyValue, "_status", translate("当前运行状态"))
o.rawhtml = true
o.cfgvalue = function(self, section)
	if is_running then
		return string.format([[<span style="color: #10b981; font-weight: bold;">● %s</span> (监听端口: %s)]],
			translate("正在运行中"), port)
	else
		return string.format([[<span style="color: #ef4444; font-weight: bold;">● %s</span>]],
			translate("已停止"))
	end
end

-- 打开 Web 后台按钮
o = s:option(DummyValue, "_open", translate("Web 管理后台"))
o.rawhtml = true
o.cfgvalue = function(self, section)
	local url = string.format("http://%s:%s/", host, port)
	return string.format([[<a href="%s" target="_blank" class="cbi-button cbi-button-apply" style="display:inline-block; text-decoration:none; padding: 6px 16px; font-weight: bold;">%s &gt;&gt;</a>]],
		url, translate("打开 iStore NVR 独立监控后台"))
end

s = m:section(NamedSection, "config", "istore-nvr", translate("基础运行参数配置"))
s.anonymous = true

o = s:option(Flag, "enabled", translate("启用 iStore NVR 服务"))
o.default = "1"
o.rmempty = false

o = s:option(Value, "port", translate("Web 服务端口"))
o.datatype = "port"
o.default = "8080"
o.rmempty = false

o = s:option(Value, "data_dir", translate("数据与数据库存储目录"))
o.default = "/mnt/sata1-4/istore-nvr/data"
o.description = translate("安全保护：严禁使用 / 或 /overlay 根目录，请确保设置为本地 SATA/NVMe 或外接存储目录")
o.rmempty = false

o = s:option(Value, "record_dir", translate("录像存储主目录"))
o.default = "/mnt/sata1-4/recordings"
o.description = translate("摄像头本地切片录像保存路径")
o.rmempty = false

function m.on_after_commit(self)
	sys.call("/etc/init.d/istore-nvr restart >/dev/null 2>&1")
end

return m
