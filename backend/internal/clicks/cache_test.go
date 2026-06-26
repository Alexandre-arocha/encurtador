package clicks

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRawIPCacheExpiresEntries(t *testing.T) {
	clickID := uuid.MustParse("018ff7d0-9c7a-7a9a-a85f-0d82b8c08c1d")
	cache := NewRawIPCache(time.Nanosecond)
	cache.Put(clickID, "203.0.113.10")

	time.Sleep(time.Millisecond)

	if got, ok := cache.Get(clickID); ok || got != "" {
		t.Fatalf("Get = (%q, %v), want expired miss", got, ok)
	}
}
