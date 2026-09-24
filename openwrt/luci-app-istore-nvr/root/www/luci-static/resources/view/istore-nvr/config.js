'use strict';
'require view';
'require fs';
'require ui';
'require uci';
'require form';
'require poll';

async function checkProcess() {
	try {
		const res = await fs.exec('/bin/pidof', ['istore-nvr']);
		if (res.code === 0 && res.stdout.trim() !== '') {
			return { running: true, pid: res.stdout.trim() };
		}
	} catch (err) {}
	return { running: false, pid: null };
}

function renderStatusBar(status, port) {
	var isRunning = status.running;
	var statusText = isRunning ? _('正在运行中') : _('未运行');
	var color = isRunning ? '#10b981' : '#ef4444';
	var icon = isRunning ? '●' : '○';

	var html = String.format('<em><span style="color:%s; font-size:14px; font-weight:bold;">%s %s %s</span></em>',
		color, icon, _('NVR摄像头管理服务'), statusText);

	if (isRunning && status.pid) {
		html += ' <small style="color:#64748b;">(PID: ' + status.pid + ')</small>';
	}

	if (isRunning) {
		var host = window.location.hostname;
		var fullUrl = 'http://' + host + ':' + port + '/';
		html += String.format('&#160;&#160;<a class="btn cbi-button cbi-button-apply" style="display:inline-block; font-weight:bold; background-color:#3b82f6; color:#ffffff; padding:5px 16px; text-decoration:none; border-radius:4px;" href="%s" target="_blank">%s &gt;&gt;</a>',
			fullUrl, _('进入 NVR Web 监控后台'));
	}

	return html;
}

return view.extend({
	load: function() {
		return Promise.all([
			uci.load('istore-nvr')
		]);
	},

	render: function() {
		var m, s, o;
		var port = uci.get('istore-nvr', 'config', 'port') || '8080';

		m = new form.Map('istore-nvr', _('NVR摄像头管理服务'),
			_('专为软路由定制的高性能网络视频录像系统。支持海康威视等摄像头跨网段直通录像、WebRTC 超低延迟多画面实时预览及局域网 NAS (SMB/CIFS) 网络存储。'));

		s = m.section(form.TypedSection);
		s.anonymous = true;
		s.render = function() {
			var statusView = E('p', { id: 'nvr_status' }, '<span class="spinning"></span> ' + _('正在检测服务运行状态...'));
			poll.add(function() {
				return checkProcess().then(function(res) {
					statusView.innerHTML = renderStatusBar(res, port);
				}).catch(function(err) {
					statusView.innerHTML = '<span style="color:orange;">⚠ ' + _('状态检测异常') + '</span>';
				});
			}, 3);

			return E('div', { class: 'cbi-section', id: 'status_bar', style: 'padding: 14px; background: rgba(0,0,0,0.03); border-radius: 6px; margin-bottom: 16px;' }, [
				statusView
			]);
		};

		s = m.section(form.NamedSection, 'config', 'istore-nvr', _('基础运行参数配置'));

		o = s.option(form.Flag, 'enabled', _('启用服务'));
		o.default = o.enabled;
		o.rmempty = false;

		o = s.option(form.Value, 'port', _('Web 管理服务监听端口'));
		o.datatype = 'port';
		o.default = '8080';
		o.rmempty = false;

		o = s.option(form.Value, 'data_dir', _('数据与数据库存储路径'));
		o.default = '/mnt/sata1-4/istore-nvr/data';
		o.description = _('安全规范：严禁使用 / 或 /overlay 根分区，请务必指定到本地 SATA/NVMe 数据盘或外部存储');
		o.rmempty = false;

		o = s.option(form.Value, 'record_dir', _('本地录像切片存储目录'));
		o.default = '/mnt/sata1-4/recordings';
		o.description = _('切片视频文件落盘根路径');
		o.rmempty = false;

		return m.render();
	}
});
