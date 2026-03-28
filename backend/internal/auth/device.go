package auth

import (
	"regexp"
	"strings"

	"hearth/internal/models"
)

// ParseUserAgent extracts device information from a User-Agent string
func ParseUserAgent(userAgent string) models.DeviceInfo {
	info := models.DeviceInfo{
		DeviceType: models.DeviceTypeUnknown,
	}

	if userAgent == "" {
		info.DeviceName = "Unknown Device"
		return info
	}

	// Detect device type
	info.DeviceType = detectDeviceType(userAgent)

	// Detect browser and version
	info.Browser, info.BrowserVersion = detectBrowser(userAgent)

	// Detect OS and version
	info.OS, info.OSVersion = detectOS(userAgent)

	// Build device name
	if info.Browser != "" && info.OS != "" {
		info.DeviceName = info.Browser + " on " + info.OS
	} else if info.Browser != "" {
		info.DeviceName = info.Browser
	} else if info.OS != "" {
		info.DeviceName = info.OS
	} else {
		info.DeviceName = "Unknown Device"
	}

	return info
}

func detectDeviceType(ua string) models.DeviceType {
	ua = strings.ToLower(ua)

	// Mobile patterns
	mobilePatterns := []string{
		"mobile", "android", "iphone", "ipod", "blackberry",
		"windows phone", "webos", "opera mini", "opera mobi",
	}
	for _, pattern := range mobilePatterns {
		if strings.Contains(ua, pattern) {
			// Check if it's a tablet
			if strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet") {
				return models.DeviceTypeTablet
			}
			// Android tablets often have "android" but not "mobile"
			if strings.Contains(ua, "android") && !strings.Contains(ua, "mobile") {
				return models.DeviceTypeTablet
			}
			return models.DeviceTypeMobile
		}
	}

	// Tablet patterns
	tabletPatterns := []string{"ipad", "tablet", "kindle", "silk", "playbook"}
	for _, pattern := range tabletPatterns {
		if strings.Contains(ua, pattern) {
			return models.DeviceTypeTablet
		}
	}

	// Default to desktop
	return models.DeviceTypeDesktop
}

func detectBrowser(ua string) (name, version string) {
	// Order matters - more specific patterns first
	browsers := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"Edge", regexp.MustCompile(`(?i)Edg(?:e|A|iOS)?[/\s](\d+(?:\.\d+)*)`)},
		{"Opera", regexp.MustCompile(`(?i)(?:OPR|Opera)[/\s](\d+(?:\.\d+)*)`)},
		{"Chrome", regexp.MustCompile(`(?i)Chrome[/\s](\d+(?:\.\d+)*)`)},
		{"Firefox", regexp.MustCompile(`(?i)Firefox[/\s](\d+(?:\.\d+)*)`)},
		{"Safari", regexp.MustCompile(`(?i)Version[/\s](\d+(?:\.\d+)*).*Safari`)},
		{"Safari", regexp.MustCompile(`(?i)Safari[/\s](\d+(?:\.\d+)*)`)},
		{"Internet Explorer", regexp.MustCompile(`(?i)(?:MSIE\s|Trident.*rv:)(\d+(?:\.\d+)*)`)},
	}

	for _, browser := range browsers {
		matches := browser.pattern.FindStringSubmatch(ua)
		if len(matches) >= 2 {
			return browser.name, matches[1]
		}
	}

	return "", ""
}

func detectOS(ua string) (name, version string) {
	osPatterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"iOS", regexp.MustCompile(`(?i)(?:iPhone|iPad|iPod).*OS\s(\d+(?:[_\.]\d+)*)`)},
		{"macOS", regexp.MustCompile(`(?i)Mac OS X\s(\d+(?:[_\.]\d+)*)`)},
		{"macOS", regexp.MustCompile(`(?i)Macintosh`)},
		{"Android", regexp.MustCompile(`(?i)Android\s(\d+(?:\.\d+)*)`)},
		{"Windows", regexp.MustCompile(`(?i)Windows NT\s(\d+(?:\.\d+)*)`)},
		{"Windows", regexp.MustCompile(`(?i)Windows\s(\d+(?:\.\d+)*)`)},
		{"Linux", regexp.MustCompile(`(?i)Linux`)},
		{"Chrome OS", regexp.MustCompile(`(?i)CrOS`)},
	}

	for _, os := range osPatterns {
		matches := os.pattern.FindStringSubmatch(ua)
		if len(matches) >= 1 {
			ver := ""
			if len(matches) >= 2 {
				ver = strings.ReplaceAll(matches[1], "_", ".")
			}

			// Convert Windows NT version to friendly name
			if os.name == "Windows" && ver != "" {
				ver = windowsVersionToName(ver)
			}

			return os.name, ver
		}
	}

	return "", ""
}

func windowsVersionToName(ntVersion string) string {
	versions := map[string]string{
		"10.0": "10/11", // Can't distinguish 10 from 11 by NT version
		"6.3":  "8.1",
		"6.2":  "8",
		"6.1":  "7",
		"6.0":  "Vista",
		"5.2":  "XP x64",
		"5.1":  "XP",
	}

	if name, ok := versions[ntVersion]; ok {
		return name
	}
	return ntVersion
}

// GetClientIP extracts the real client IP from headers
func GetClientIP(remoteAddr string, xForwardedFor, xRealIP string) string {
	// Prefer X-Forwarded-For (first IP in list)
	if xForwardedFor != "" {
		parts := strings.Split(xForwardedFor, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Fall back to X-Real-IP
	if xRealIP != "" {
		return strings.TrimSpace(xRealIP)
	}

	// Fall back to remote address
	// Remove port if present
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		// Check if it's an IPv6 address
		if strings.Count(remoteAddr, ":") > 1 {
			// IPv6 - handle [::1]:port format
			if strings.HasPrefix(remoteAddr, "[") {
				if endBracket := strings.Index(remoteAddr, "]"); endBracket != -1 {
					return remoteAddr[1:endBracket]
				}
			}
			return remoteAddr
		}
		return remoteAddr[:idx]
	}

	return remoteAddr
}
