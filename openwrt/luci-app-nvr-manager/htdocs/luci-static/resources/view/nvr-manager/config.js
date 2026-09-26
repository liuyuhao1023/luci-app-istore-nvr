'use strict';
'require view';
'require fs';
'require ui';
'require uci';
'require form';
'require poll';

function checkProcess() {
	return fs.exec('/etc/init.d/nvr-manager', ['running']).then(function(res) {
		return { running: (res.code === 0) };
	}).catch(function() {
		return { running: false };
	});
}

function renderStatusHeader(status, port) {
	var isRunning = status.running;
	var statusText = isRunning ? _('正在运行中') : _('未运行');
	var statusColor = isRunning ? '#10b981' : '#ef4444';
	var icon = isRunning ? '●' : '○';

	var host = window.location.hostname;
	var fullUrl = window.location.protocol + '//' + host + ':' + port + '/';

	var btnHtml = '';
	if (isRunning) {
		btnHtml = String.format(
			'<a class="btn cbi-button cbi-button-apply" style="display:inline-flex; align-items:center; font-weight:bold; background-color:#2563eb; color:#ffffff; padding:8px 20px; font-size:14px; text-decoration:none; border-radius:6px; box-shadow: 0 2px 4px rgba(37,99,235,0.3);" href="%s" target="_blank">' +
			'🚀 %s &gt;&gt;</a>' +
			'&#160;&#160;' +
			'<a class="btn cbi-button" style="display:inline-flex; align-items:center; padding:8px 16px; font-size:14px; text-decoration:none; border-radius:6px;" href="%s">' +
			'🖥️ %s</a>',
			fullUrl, _('打开 Web 监控中心'),
			L.url('admin', 'services', 'nvr-manager', 'web'), _('嵌入式预览')
		);
	} else {
		btnHtml = String.format(
			'<span style="color:#ef4444; font-size:13px;">%s</span>',
			_('服务未运行。请确保已指定外部存储路径并勾选【启用服务】后保存应用。')
		);
	}

	return String.format(
		'<div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 18px 20px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.05);">' +
			'<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 12px;">' +
				'<div>' +
					'<div style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px;">' +
						'<span style="color:%s; font-size:16px;">%s</span>' +
						'<span style="font-size: 16px; font-weight: bold; color: #1e293b;">%s</span>' +
						'<span style="background: %s15; color: %s; padding: 2px 10px; border-radius: 9999px; font-size: 12px; font-weight: 600;">%s</span>' +
					'</div>' +
					'<div style="font-size: 13px; color: #64748b;">' +
						'%s <a href="%s" target="_blank" style="color: #2563eb; font-weight: 600; text-decoration: underline;">%s</a>' +
					'</div>' +
				'</div>' +
				'<div>' +
					'%s' +
				'</div>' +
			'</div>' +
		'</div>',
		statusColor, icon,
		_('NVR 监控管理服务'),
		statusColor, statusColor, statusText,
		_('Web 控制台访问地址:'), fullUrl, fullUrl,
		btnHtml
	);
}

return view.extend({
	load: function() {
		return Promise.all([
			uci.load('nvr-manager')
		]);
	},

	render: function() {
		var m, s, o;
		var port = uci.get('nvr-manager', 'config', 'port') || '8080';

		m = new form.Map('nvr-manager', _('NVR 视频监控管理'),
			_('专为软路由定制的高性能网络视频监控系统。支持海康威视等摄像头跨网段直通录像、超低延迟多画面实时预览及局域网 NAS (SMB/CIFS) 网络存储。'));

		s = m.section(form.TypedSection);
		s.anonymous = true;
		s.render = function() {
			var statusView = E('div', { id: 'nvr_status_container' }, [
				E('p', { class: 'spinning' }, _('正在检测服务运行状态...'))
			]);

			poll.add(function() {
				return checkProcess().then(function(res) {
					statusView.innerHTML = renderStatusHeader(res, port);
				}).catch(function(err) {
					statusView.innerHTML = '<div style="color:orange; padding:10px;">⚠ ' + _('状态检测异常') + '</div>';
				});
			}, 3);

			return statusView;
		};

		s = m.section(form.NamedSection, 'config', 'nvr-manager', _('基础运行参数配置'));

		o = s.option(form.Flag, 'enabled', _('启用服务'));
		o.default = o.disabled;
		o.rmempty = false;

		o = s.option(form.Value, 'port', _('Web 管理服务监听端口'));
		o.datatype = 'port';
		o.default = '8080';
		o.rmempty = false;

		o = s.option(form.Value, 'data_dir', _('数据与数据库存储路径'));
		o.placeholder = '/mnt/sda1/nvr-manager/data';
		o.description = _('安全规范：严禁使用 / 或 /overlay 根分区，请指定到本地 SATA/NVMe 数据盘或外部存储路径。未配置时服务不会启动。');
		o.rmempty = false;

		o = s.option(form.Value, 'record_dir', _('本地录像切片存储目录'));
		o.placeholder = '/mnt/sda1/recordings';
		o.description = _('切片视频文件落盘根路径，建议指定到大容量硬盘挂载点。');
		o.rmempty = true;

		return m.render();
	}
});
