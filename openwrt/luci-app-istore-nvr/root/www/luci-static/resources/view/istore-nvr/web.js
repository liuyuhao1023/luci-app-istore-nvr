'use strict';
'require view';
'require uci';

return view.extend({
	load: function() {
		return uci.load('istore-nvr');
	},

	render: function() {
		var port = uci.get('istore-nvr', 'config', 'port') || '8080';
		var host = window.location.hostname;
		var url = 'http://' + host + ':' + port + '/';

		return E('div', { 'style': 'width: 100%; height: calc(100vh - 140px); display: flex; flex-direction: column;' }, [
			E('div', { 'style': 'margin-bottom: 10px; display: flex; justify-content: space-between; align-items: center; background: rgba(0,0,0,0.03); padding: 8px 14px; border-radius: 6px;' }, [
				E('span', { 'style': 'font-size: 13px; color: #475569;' }, [
					E('strong', {}, _('服务运行地址: ')),
					E('span', { 'style': 'font-family: monospace; color: #2563eb;' }, url)
				]),
				E('a', {
					'class': 'btn cbi-button cbi-button-apply',
					'href': url,
					'target': '_blank',
					'style': 'padding: 4px 14px; font-weight: bold; background-color: #3b82f6; color: #ffffff; text-decoration: none; border-radius: 4px;'
				}, _('在新标签页中全屏打开 NVR 后台 >>'))
			]),
			E('iframe', {
				'src': url,
				'style': 'width: 100%; flex: 1; border: 1px solid #cbd5e1; border-radius: 6px; background-color: #0b0f17;',
				'allow': 'autoplay; fullscreen'
			})
		]);
	}
});
