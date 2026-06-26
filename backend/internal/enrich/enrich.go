// Package enrich transforma um clique cru nos campos derivados usados pelo
// dashboard. GeoIP e user-agent sao best-effort: entrada ruim deixa campos
// vazios, sem transformar o enriquecimento em erro fatal.
package enrich

import (
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"

	"github.com/mileusna/useragent"
	"github.com/oschwald/geoip2-golang"
)

const (
	DeviceMobile  = "mobile"
	DeviceTablet  = "tablet"
	DeviceDesktop = "desktop"
	DeviceBot     = "bot"
)

// Raw representa os dados brutos coletados no redirect. IP pode ser usado para
// lookup GeoIP, mas nunca deve ser persistido cru.
type Raw struct {
	UARaw    string
	IP       string
	Referrer string
}

// Enriched concentra os campos derivados persistidos em clicks.
//
// Country usa ISO-3166 alpha-2 em maiusculas quando o GeoLite2 retornar pais.
type Enriched struct {
	DeviceType string
	Browser    string
	OS         string
	Country    string
	City       string
}

// GeoResult e a parte de geo que o modulo precisa.
type GeoResult struct {
	Country string
	City    string
}

// GeoLookup permite testar o enriquecimento sem depender de arquivo .mmdb.
type GeoLookup interface {
	LookupCity(ip netip.Addr) (GeoResult, bool)
}

// Service permite injetar o lookup de geo. Um lookup nil deixa Country/City
// vazios e ainda processa user-agent normalmente.
type Service struct {
	geo GeoLookup
}

// New cria um enriquecedor com lookup de geo opcional.
func New(geo GeoLookup) *Service {
	return &Service{geo: geo}
}

// Enrich executa o enriquecimento usando o GeoLite2 apontado por GEOLITE_DB_PATH
// quando disponivel. Falha ao abrir/consultar geo e tratada como best-effort.
func Enrich(raw Raw) Enriched {
	return New(defaultGeo()).Enrich(raw)
}

// Enrich executa o parse de UA e o lookup de geo configurado no Service.
func (s *Service) Enrich(raw Raw) Enriched {
	out := enrichUA(raw.UARaw)

	if s == nil || s.geo == nil {
		return out
	}
	ip, ok := publicAddr(raw.IP)
	if !ok {
		return out
	}
	geo, ok := s.geo.LookupCity(ip)
	if !ok {
		return out
	}

	out.Country = strings.ToUpper(strings.TrimSpace(geo.Country))
	out.City = strings.TrimSpace(geo.City)
	return out
}

func enrichUA(raw string) Enriched {
	ua := useragent.Parse(raw)
	out := Enriched{
		Browser: strings.TrimSpace(ua.Name),
		OS:      strings.TrimSpace(ua.OS),
	}

	switch {
	case ua.Bot:
		out.DeviceType = DeviceBot
	case ua.Tablet:
		out.DeviceType = DeviceTablet
	case ua.Mobile:
		out.DeviceType = DeviceMobile
	case ua.Desktop:
		out.DeviceType = DeviceDesktop
	}

	return out
}

func publicAddr(raw string) (netip.Addr, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return netip.Addr{}, false
	}
	if comma := strings.IndexByte(raw, ','); comma >= 0 {
		raw = strings.TrimSpace(raw[:comma])
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}

	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Addr{}, false
	}
	addr = addr.Unmap()
	if !addr.IsValid() ||
		!addr.IsGlobalUnicast() ||
		addr.IsPrivate() ||
		addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified() {
		return netip.Addr{}, false
	}
	return addr, true
}

var (
	defaultGeoOnce     sync.Once
	defaultGeoResolver GeoLookup
)

func defaultGeo() GeoLookup {
	defaultGeoOnce.Do(func() {
		path := strings.TrimSpace(os.Getenv("GEOLITE_DB_PATH"))
		if path == "" {
			return
		}
		resolver, err := OpenMaxMind(path)
		if err != nil {
			return
		}
		defaultGeoResolver = resolver
	})
	return defaultGeoResolver
}

// MaxMindGeo adapta o reader GeoLite2-City ao contrato GeoLookup.
type MaxMindGeo struct {
	reader *geoip2.Reader
}

// OpenMaxMind abre um arquivo GeoLite2-City local.
func OpenMaxMind(path string) (*MaxMindGeo, error) {
	reader, err := geoip2.Open(path)
	if err != nil {
		return nil, err
	}
	return &MaxMindGeo{reader: reader}, nil
}

// LookupCity resolve pais/cidade para IP publico. Falhas viram retorno false.
func (g *MaxMindGeo) LookupCity(ip netip.Addr) (GeoResult, bool) {
	if g == nil || g.reader == nil {
		return GeoResult{}, false
	}

	record, err := g.reader.City(net.IP(ip.AsSlice()))
	if err != nil || record == nil {
		return GeoResult{}, false
	}

	country := record.Country.IsoCode
	if country == "" {
		country = record.RegisteredCountry.IsoCode
	}
	city := record.City.Names["pt-BR"]
	if city == "" {
		city = record.City.Names["en"]
	}

	result := GeoResult{
		Country: strings.ToUpper(strings.TrimSpace(country)),
		City:    strings.TrimSpace(city),
	}
	return result, result.Country != "" || result.City != ""
}

// Close fecha o reader MaxMind quando ele foi aberto pelo caller.
func (g *MaxMindGeo) Close() error {
	if g == nil || g.reader == nil {
		return nil
	}
	return g.reader.Close()
}
