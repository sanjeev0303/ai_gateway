package main

import (
	"fmt"
	"os"

	"ai-governance-saas/backend/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	redacted := cfg.Redacted()
	fmt.Printf("Configuration valid (env=%s, addr=%s, db_enabled=%v, redis_enabled=%v, eventbus=%s)\n",
		redacted.Env, redacted.Server.Addr, redacted.Database.Enabled, redacted.Redis.Enabled, redacted.Eventbus.Kind)
}
