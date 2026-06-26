package clicks

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultRawIPTTL = 5 * time.Minute

type rawIPEntry struct {
	value     string
	expiresAt time.Time
}

// RawIPCache guarda IP cru apenas em memoria e por pouco tempo. Ele existe para
// permitir GeoIP assíncrono sem persistir IP cru em clicks nem em jobs River.
type RawIPCache struct {
	ttl     time.Duration
	entries sync.Map
}

// NewRawIPCache cria um cache volatil para IPs crus.
func NewRawIPCache(ttl time.Duration) *RawIPCache {
	if ttl <= 0 {
		ttl = defaultRawIPTTL
	}
	return &RawIPCache{ttl: ttl}
}

func (c *RawIPCache) Put(clickID uuid.UUID, rawIP string) {
	if c == nil || rawIP == "" {
		return
	}
	c.entries.Store(clickID, rawIPEntry{
		value:     rawIP,
		expiresAt: time.Now().Add(c.ttl),
	})
}

func (c *RawIPCache) Get(clickID uuid.UUID) (string, bool) {
	if c == nil {
		return "", false
	}
	value, ok := c.entries.Load(clickID)
	if !ok {
		return "", false
	}
	entry, ok := value.(rawIPEntry)
	if !ok || time.Now().After(entry.expiresAt) {
		c.entries.Delete(clickID)
		return "", false
	}
	return entry.value, true
}

func (c *RawIPCache) Delete(clickID uuid.UUID) {
	if c == nil {
		return
	}
	c.entries.Delete(clickID)
}
