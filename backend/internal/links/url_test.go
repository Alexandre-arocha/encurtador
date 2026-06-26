package links

import (
	"errors"
	"testing"
)

func TestValidateTargetURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "https public host", raw: "https://example.com/path"},
		{name: "http public ip", raw: "http://8.8.8.8/dns-query"},
		{name: "empty", raw: "", wantErr: true},
		{name: "unsupported scheme", raw: "ftp://example.com/file", wantErr: true},
		{name: "missing host", raw: "https:///sem-host", wantErr: true},
		{name: "localhost", raw: "http://localhost:3000", wantErr: true},
		{name: "localhost suffix", raw: "http://app.localhost", wantErr: true},
		{name: "loopback ipv4", raw: "http://127.0.0.1:8080", wantErr: true},
		{name: "private ipv4", raw: "http://192.168.1.10", wantErr: true},
		{name: "link local ipv4", raw: "http://169.254.10.10", wantErr: true},
		{name: "loopback ipv6", raw: "http://[::1]:8080", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTargetURL(tt.raw)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("ValidateTargetURL(%q) = %v, want nil", tt.raw, err)
				}
				return
			}
			if !errors.Is(err, ErrInvalidURL) {
				t.Fatalf("ValidateTargetURL(%q) = %v, want ErrInvalidURL", tt.raw, err)
			}
		})
	}
}
