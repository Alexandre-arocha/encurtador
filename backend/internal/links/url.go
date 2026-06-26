package links

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateTargetURL garante que a URL de destino é http/https e não aponta para
// localhost ou para um endereço de rede privada/loopback/link-local.
//
// Observação: a checagem é feita sobre o host literal da URL (não há resolução
// de DNS aqui). Isso barra os casos óbvios de SSRF por IP/localhost; uma
// proteção completa exigiria resolver o host no momento do redirect.
func ValidateTargetURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("%w: a URL é obrigatória", ErrInvalidURL)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: não foi possível interpretar a URL", ErrInvalidURL)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: o scheme deve ser http ou https", ErrInvalidURL)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("%w: host ausente", ErrInvalidURL)
	}

	lhost := strings.ToLower(host)
	if lhost == "localhost" || strings.HasSuffix(lhost, ".localhost") {
		return fmt.Errorf("%w: localhost não é permitido", ErrInvalidURL)
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("%w: endereços privados ou de loopback não são permitidos", ErrInvalidURL)
		}
	}

	return nil
}
