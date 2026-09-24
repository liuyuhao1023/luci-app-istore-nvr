module("luci.controller.istore-nvr", package.seeall)

function index()
	if not nixio.fs.access("/etc/config/istore-nvr") then
		return
	end

	entry({"admin", "services", "istore-nvr"}, cbi("istore-nvr"), _("iStore NVR 监控"), 35).dependent = true
	entry({"admin", "services", "istore-nvr", "status"}, call("act_status")).leaf = true
end

function act_status()
	local sys = require "luci.sys"
	local uci = require "luci.model.uci".cursor()
	local port = tonumber(uci:get("istore-nvr", "config", "port")) or 8080

	local is_running = (sys.call("pidof istore-nvr >/dev/null") == 0)

	luci.http.prepare_content("application/json")
	luci.http.write_json({
		running = is_running,
		port = port
	})
end
