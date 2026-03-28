package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

func TestParseUserAgent_Chrome_Windows(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	info := ParseUserAgent(ua)

	assert.Equal(t, "Chrome", info.Browser)
	assert.Equal(t, "120.0.0.0", info.BrowserVersion)
	assert.Equal(t, "Windows", info.OS)
	assert.Equal(t, "10/11", info.OSVersion)
	assert.Equal(t, models.DeviceTypeDesktop, info.DeviceType)
	assert.Equal(t, "Chrome on Windows", info.DeviceName)
}

func TestParseUserAgent_Safari_macOS(t *testing.T) {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15"

	info := ParseUserAgent(ua)

	assert.Equal(t, "Safari", info.Browser)
	assert.Equal(t, "17.0", info.BrowserVersion)
	assert.Equal(t, "macOS", info.OS)
	assert.Equal(t, "10.15.7", info.OSVersion)
	assert.Equal(t, models.DeviceTypeDesktop, info.DeviceType)
}

func TestParseUserAgent_Firefox_Linux(t *testing.T) {
	ua := "Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0"

	info := ParseUserAgent(ua)

	assert.Equal(t, "Firefox", info.Browser)
	assert.Equal(t, "121.0", info.BrowserVersion)
	assert.Equal(t, "Linux", info.OS)
	assert.Equal(t, models.DeviceTypeDesktop, info.DeviceType)
}

func TestParseUserAgent_Chrome_Android_Mobile(t *testing.T) {
	ua := "Mozilla/5.0 (Linux; Android 13; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.43 Mobile Safari/537.36"

	info := ParseUserAgent(ua)

	assert.Equal(t, "Chrome", info.Browser)
	assert.Equal(t, "Android", info.OS)
	assert.Equal(t, "13", info.OSVersion)
	assert.Equal(t, models.DeviceTypeMobile, info.DeviceType)
}

func TestParseUserAgent_Safari_iOS(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1"

	info := ParseUserAgent(ua)

	assert.Equal(t, "Safari", info.Browser)
	assert.Equal(t, "iOS", info.OS)
	assert.Equal(t, "17.1.1", info.OSVersion)
	assert.Equal(t, models.DeviceTypeMobile, info.DeviceType)
}

func TestParseUserAgent_Safari_iPad(t *testing.T) {
	ua := "Mozilla/5.0 (iPad; CPU OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1"

	info := ParseUserAgent(ua)

	assert.Equal(t, "Safari", info.Browser)
	assert.Equal(t, "iOS", info.OS)
	assert.Equal(t, models.DeviceTypeTablet, info.DeviceType)
}

func TestParseUserAgent_Android_Tablet(t *testing.T) {
	ua := "Mozilla/5.0 (Linux; Android 13; SM-T870) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.43 Safari/537.36"

	info := ParseUserAgent(ua)

	assert.Equal(t, "Chrome", info.Browser)
	assert.Equal(t, "Android", info.OS)
	// Android tablet without "mobile" should be detected as tablet
	assert.Equal(t, models.DeviceTypeTablet, info.DeviceType)
}

func TestParseUserAgent_Edge_Windows(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0"

	info := ParseUserAgent(ua)

	assert.Equal(t, "Edge", info.Browser)
	assert.Equal(t, "120.0.0.0", info.BrowserVersion)
	assert.Equal(t, "Windows", info.OS)
	assert.Equal(t, models.DeviceTypeDesktop, info.DeviceType)
}

func TestParseUserAgent_Opera(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 OPR/106.0.0.0"

	info := ParseUserAgent(ua)

	assert.Equal(t, "Opera", info.Browser)
	assert.Equal(t, "106.0.0.0", info.BrowserVersion)
}

func TestParseUserAgent_Empty(t *testing.T) {
	info := ParseUserAgent("")

	assert.Equal(t, "Unknown Device", info.DeviceName)
	assert.Equal(t, models.DeviceTypeUnknown, info.DeviceType)
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	ip := GetClientIP("127.0.0.1:8080", "203.0.113.195, 70.41.3.18, 150.172.238.178", "")

	assert.Equal(t, "203.0.113.195", ip)
}

func TestGetClientIP_XRealIP(t *testing.T) {
	ip := GetClientIP("127.0.0.1:8080", "", "203.0.113.195")

	assert.Equal(t, "203.0.113.195", ip)
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	ip := GetClientIP("192.168.1.100:54321", "", "")

	assert.Equal(t, "192.168.1.100", ip)
}

func TestGetClientIP_IPv6(t *testing.T) {
	ip := GetClientIP("[::1]:8080", "", "")

	assert.Equal(t, "::1", ip)
}

func TestGetClientIP_Priority(t *testing.T) {
	// X-Forwarded-For takes priority over X-Real-IP
	ip := GetClientIP("127.0.0.1:8080", "10.0.0.1", "20.0.0.1")

	assert.Equal(t, "10.0.0.1", ip)
}
