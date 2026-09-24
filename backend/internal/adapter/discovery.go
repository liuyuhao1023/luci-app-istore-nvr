package adapter

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DiscoveryEngine 跨网段与本网段摄像头发现引擎
type DiscoveryEngine struct{}

func NewDiscoveryEngine() *DiscoveryEngine {
	return &DiscoveryEngine{}
}

// ONVIF WS-Discovery Probe SOAP 模板
const wsDiscoveryProbeXML = `<?xml version="1.0" encoding="utf-8"?>
<Envelope xmlns:tds="http://www.onvif.org/ver10/device/wsdl" xmlns="http://www.w3.org/2003/05/soap-envelope">
  <Header xmlns:wsa="http://schemas.xmlsoap.org/ws/2004/08/addressing">
    <wsa:MessageID>uuid:84ede3de-7dec-11d0-c360-f01234567890</wsa:MessageID>
    <wsa:To>urn:schemas-xmlsoap-org:ws:2005/04/discovery</wsa:To>
    <wsa:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</wsa:Action>
  </Header>
  <Body>
    <Probe xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns="http://schemas.xmlsoap.org/ws/2005/04/discovery">
      <Types>tds:Device</Types>
    </Probe>
  </Body>
</Envelope>`

// ProbeMulticast 执行本地局域网 ONVIF 组播自动发现
func (d *DiscoveryEngine) ProbeMulticast(timeoutSec int) ([]DiscoveredDevice, error) {
	if timeoutSec <= 0 {
		timeoutSec = 3
	}

	multicastAddr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:3702")
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(time.Duration(timeoutSec) * time.Second))

	// 发送组播 Probe
	if _, err := conn.WriteTo([]byte(wsDiscoveryProbeXML), multicastAddr); err != nil {
		return nil, err
	}

	results := make(map[string]DiscoveredDevice)
	buf := make([]byte, 8192)

	for {
		n, src, err := conn.ReadFrom(buf)
		if err != nil {
			break // 超时正常退出
		}

		ipStr := src.(*net.UDPAddr).IP.String()
		dev := parseWSDiscoveryResponse(buf[:n], ipStr, "onvif_multicast")
		if dev != nil {
			results[dev.IP] = *dev
		}
	}

	var list []DiscoveredDevice
	for _, dev := range results {
		list = append(list, dev)
	}
	return list, nil
}

// ScanSubnets 扫描用户指定的多个网段 (CIDR 或 IP范围)，跨网段探测摄像头
func (d *DiscoveryEngine) ScanSubnets(ctx context.Context, cidrList []string, concurrency int) ([]DiscoveredDevice, error) {
	if concurrency <= 0 {
		concurrency = 30
	}

	var targetIPs []string
	for _, item := range cidrList {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		ips, err := parseIPRangeOrCIDR(item)
		if err == nil {
			targetIPs = append(targetIPs, ips...)
		}
	}

	if len(targetIPs) == 0 {
		return nil, fmt.Errorf("未解析到有效的扫描目标IP")
	}

	ipChan := make(chan string, len(targetIPs))
	for _, ip := range targetIPs {
		ipChan <- ip
	}
	close(ipChan)

	var mu sync.Mutex
	var discovered []DiscoveredDevice
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range ipChan {
				select {
				case <-ctx.Done():
					return
				default:
					if dev := d.probeSingleTarget(ip); dev != nil {
						mu.Lock()
						discovered = append(discovered, *dev)
						mu.Unlock()
					}
				}
			}
		}()
	}

	wg.Wait()
	return discovered, nil
}

// probeSingleTarget 对单个跨网段 IP 进行多协议探测
func (d *DiscoveryEngine) probeSingleTarget(ip string) *DiscoveredDevice {
	timeout := 1200 * time.Millisecond

	// 1. 优先探测标准 RTSP 554 端口
	rtspOpen := isTCPPortOpen(ip, 554, timeout)
	httpOpen := isTCPPortOpen(ip, 80, timeout)
	hikServiceOpen := isTCPPortOpen(ip, 8000, timeout)

	if !rtspOpen && !httpOpen && !hikServiceOpen {
		return nil
	}

	brand := "unknown"
	method := "port_scan"
	modelName := ""

	if hikServiceOpen {
		brand = "hikvision"
		method = "hik_port_8000"
	}

	// 2. 发送单播 ONVIF Probe 验证
	if unicastDev := d.probeUnicastONVIF(ip, timeout); unicastDev != nil {
		return unicastDev
	}

	if rtspOpen && brand == "unknown" {
		brand = "hikvision" // 海康第一阶段适配默认判定
		modelName = "网络摄像头 (RTSP 554)"
	}

	return &DiscoveredDevice{
		IP:          ip,
		Port:        554,
		Brand:       brand,
		Model:       modelName,
		ProbeMethod: method,
	}
}

// probeUnicastONVIF 发送单播 UDP 3702 探测跨网段 ONVIF 设备
func (d *DiscoveryEngine) probeUnicastONVIF(ip string, timeout time.Duration) *DiscoveredDevice {
	targetAddr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(ip, "3702"))
	if err != nil {
		return nil
	}

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := conn.WriteTo([]byte(wsDiscoveryProbeXML), targetAddr); err != nil {
		return nil
	}

	buf := make([]byte, 4096)
	n, _, err := conn.ReadFrom(buf)
	if err != nil {
		return nil
	}

	return parseWSDiscoveryResponse(buf[:n], ip, "unicast_onvif")
}

// isTCPPortOpen 快速检查 TCP 端口
func isTCPPortOpen(ip string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, strconv.Itoa(port)), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// parseWSDiscoveryResponse 解析 WS-Discovery XML 报文
func parseWSDiscoveryResponse(data []byte, ip, method string) *DiscoveredDevice {
	dev := &DiscoveredDevice{
		IP:          ip,
		Port:        554,
		Brand:       "general",
		ProbeMethod: method,
	}

	raw := string(data)
	if strings.Contains(strings.ToLower(raw), "hikvision") || strings.Contains(strings.ToLower(raw), "hik") {
		dev.Brand = "hikvision"
	}

	// 简易抽取 XAddrs 地址
	if start := strings.Index(raw, "<d:XAddrs>"); start != -1 {
		end := strings.Index(raw[start:], "</d:XAddrs>")
		if end != -1 {
			xaddrs := strings.TrimSpace(raw[start+10 : start+end])
			dev.Model = fmt.Sprintf("ONVIF (%s)", xaddrs)
		}
	} else if start := strings.Index(raw, "http://"); start != -1 {
		end := strings.IndexAny(raw[start:], " \r\n<\"'")
		if end != -1 {
			dev.Model = fmt.Sprintf("ONVIF (%s)", raw[start:start+end])
		}
	}

	return dev
}

// parseIPRangeOrCIDR 解析 CIDR (如 192.168.2.0/24) 或范围 (如 192.168.2.1-192.168.2.50)
func parseIPRangeOrCIDR(input string) ([]string, error) {
	input = strings.TrimSpace(input)

	// 情况 1: CIDR
	if strings.Contains(input, "/") {
		ip, ipnet, err := net.ParseCIDR(input)
		if err != nil {
			return nil, err
		}
		var ips []string
		for curr := ip.Mask(ipnet.Mask); ipnet.Contains(curr); incIP(curr) {
			ips = append(ips, curr.String())
		}
		// 去除网络号与广播地址 (针对 /24 常见子网)
		if len(ips) > 2 {
			return ips[1 : len(ips)-1], nil
		}
		return ips, nil
	}

	// 情况 2: IP-IP 范围 (例如 192.168.2.1-192.168.2.100)
	if strings.Contains(input, "-") {
		parts := strings.Split(input, "-")
		if len(parts) == 2 {
			startIP := net.ParseIP(strings.TrimSpace(parts[0]))
			endStr := strings.TrimSpace(parts[1])
			var endIP net.IP
			if strings.Contains(endStr, ".") {
				endIP = net.ParseIP(endStr)
			} else {
				// 简写: 192.168.2.1-50
				lastDot := strings.LastIndex(parts[0], ".")
				if lastDot != -1 {
					endIP = net.ParseIP(parts[0][:lastDot+1] + endStr)
				}
			}

			if startIP != nil && endIP != nil {
				var ips []string
				curr := make(net.IP, len(startIP))
				copy(curr, startIP)
				for bytes.Compare(curr, endIP) <= 0 {
					ips = append(ips, curr.String())
					incIP(curr)
					if len(ips) > 512 { // 限制单次扫描数量防止雪崩
						break
					}
				}
				return ips, nil
			}
		}
	}

	// 情况 3: 单个 IP
	if parsed := net.ParseIP(input); parsed != nil {
		return []string{parsed.String()}, nil
	}

	return nil, fmt.Errorf("无法识别的IP或网段格式: %s", input)
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
