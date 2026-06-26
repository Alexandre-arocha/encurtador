package enrich

import (
	"net/netip"
	"testing"
)

type fakeGeo struct {
	calls int
	data  map[netip.Addr]GeoResult
}

func (f *fakeGeo) LookupCity(ip netip.Addr) (GeoResult, bool) {
	f.calls++
	result, ok := f.data[ip]
	return result, ok
}

func TestEnrichKnownUserAgents(t *testing.T) {
	tests := []struct {
		name       string
		ua         string
		deviceType string
		browser    string
		os         string
	}{
		{
			name:       "chrome desktop",
			ua:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
			deviceType: DeviceDesktop,
			browser:    "Chrome",
			os:         "Windows",
		},
		{
			name:       "safari ios",
			ua:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
			deviceType: DeviceMobile,
			browser:    "Safari",
			os:         "iOS",
		},
		{
			name:       "bot",
			ua:         "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			deviceType: DeviceBot,
			browser:    "Googlebot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(nil).Enrich(Raw{UARaw: tt.ua})
			if got.DeviceType != tt.deviceType {
				t.Fatalf("DeviceType = %q, want %q", got.DeviceType, tt.deviceType)
			}
			if got.Browser != tt.browser {
				t.Fatalf("Browser = %q, want %q", got.Browser, tt.browser)
			}
			if tt.os != "" && got.OS != tt.os {
				t.Fatalf("OS = %q, want %q", got.OS, tt.os)
			}
		})
	}
}

func TestEnrichGeoBestEffort(t *testing.T) {
	ip := netip.MustParseAddr("8.8.8.8")
	geo := &fakeGeo{data: map[netip.Addr]GeoResult{
		ip: {Country: "us", City: "Mountain View"},
	}}

	got := New(geo).Enrich(Raw{
		UARaw: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		IP:    "8.8.8.8",
	})

	if got.Country != "US" || got.City != "Mountain View" {
		t.Fatalf("geo = (%q, %q), want (US, Mountain View)", got.Country, got.City)
	}
	if geo.calls != 1 {
		t.Fatalf("geo calls = %d, want 1", geo.calls)
	}
}

func TestEnrichSkipsInvalidAndPrivateIPs(t *testing.T) {
	tests := []string{
		"",
		"not-an-ip",
		"127.0.0.1",
		"10.0.0.5",
		"172.16.0.10",
		"192.168.1.20",
		"::1",
	}

	for _, rawIP := range tests {
		t.Run(rawIP, func(t *testing.T) {
			geo := &fakeGeo{data: map[netip.Addr]GeoResult{}}
			got := New(geo).Enrich(Raw{IP: rawIP})
			if got.Country != "" || got.City != "" {
				t.Fatalf("geo = (%q, %q), want empty", got.Country, got.City)
			}
			if geo.calls != 0 {
				t.Fatalf("geo calls = %d, want 0", geo.calls)
			}
		})
	}
}
