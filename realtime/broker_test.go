package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/logger"
)

func TestBrokerCloseIsIdempotent(t *testing.T) {
	log := logger.Get(config.LoadConfig())
	b := NewBroker(func(context.Context, string) (string, error) {
		return "", nil
	}, cache.NewDevCache(log), log)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := b.Close(ctx); err != nil {
		t.Fatal(err)
	}
}
