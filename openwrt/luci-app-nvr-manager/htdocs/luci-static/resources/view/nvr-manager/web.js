'use strict';
'require view';
'require uci';

return view.extend({
	load: function() {
		return uci.load('nvr-manager');
	},

	render: function() {
		var port = uci.get('nvr-manager', 'config', 'port') || '8080';
		var host = window.location.hostname;
		var url = window.location.protocol + '//' + host + ':' + port + '/';

		var iframeId = 'nvr_web_frame_' + Math.floor(Math.random() * 100000);

		return E('div', { 'style': 'width: 100%; height: calc(100vh - 120px); min-height: 700px; display: flex; flex-direction: column; margin-top: -10px;' }, [
			E('div', {
				'style': 'margin-bottom: 12px; display: flex; justify-content: space-between; align-items: center; background: #f8fafc; border: 1px solid #e2e8f0; padding: 10px 16px; border-radius: 8px; box-shadow: 0 1px 2px rgba(0,0,0,0.05);'
			}, [
				E('div', { 'style': 'display: flex; align-items: center; gap: 12px;' }, [
					E('span', { 'style': 'font-size: 14px; font-weight: bold; color: #1e293b;' }, _('NVR 监控中心')),
					E('span', { 'style': 'font-size: 13px; color: #64748b;' }, [
						_('运行端口: '),
						E('code', { 'style': 'background:#e2e8f0; color:#0f172a; padding: 2px 6px; border-radius: 4px;' }, port)
					])
				]),
				E('div', { 'style': 'display: flex; gap: 8px;' }, [
					E('button', {
						'class': 'btn cbi-button',
						'style': 'padding: 6px 14px; border-radius: 6px; cursor: pointer;',
						'click': function() {
							var frame = document.getElementById(iframeId);
							if (frame) {
								frame.src = url;
							}
						}
					}, '🔄 ' + _('刷新界面')),
					E('a', {
						'class': 'btn cbi-button cbi-button-apply',
						'href': url,
						'target': '_blank',
						'style': 'padding: 6px 18px; font-weight: bold; background-color: #2563eb; color: #ffffff; text-decoration: none; border-radius: 6px; box-shadow: 0 2px 4px rgba(37,99,235,0.3);'
					}, '↗️ ' + _('在新窗口打开 Web >>'))
				])
			]),
			E('iframe', {
				'id': iframeId,
				'src': url,
				'style': 'width: 100%; flex: 1; border: 1px solid #cbd5e1; border-radius: 8px; background-color: #0b0f17; min-height: 640px;',
				'allow': 'autoplay; fullscreen'
			})
		]);
	}
});
