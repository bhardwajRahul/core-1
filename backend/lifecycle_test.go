package backend

import (
	"context"
	"testing"
	"time"

	"github.com/staticbackendhq/core/config"
)

func TestCloseIsIdempotent(t *testing.T) {
	Setup(config.AppConfig{
		AppEnv:           "dev",
		FromCLI:          "yes",
		DatabaseURL:      "mem",
		DataStore:        "mem",
		NoFullTextSearch: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := Close(ctx); err != nil {
		t.Fatal(err)
	}
}
