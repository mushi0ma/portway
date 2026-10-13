package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"portway/internal/config"
	"portway/internal/modelgateway"
	"portway/internal/openrouter"
	"portway/internal/problem"
	"portway/internal/secure"
)

func modelCommand(mode string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if mode == "models" {
		catalog := openrouter.New()
		defer catalog.Close()
		models, err := catalog.Models(ctx)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"policy": "free_only", "endpoint": openrouter.Endpoint, "models": models, "checked_at": time.Now().UTC()})
	}
	key, err := config.Secret("OPENROUTER_API_KEY", true)
	if err != nil {
		return err
	}
	model := os.Getenv("PORTWAY_OPENROUTER_MODEL")
	if len(os.Args) > 2 {
		model = os.Args[2]
	}
	if !openrouter.FreeID(model) {
		return errors.New("choose an explicit :free model: portway model-probe MODEL_ID (list with portway models)")
	}
	gateway, err := modelgateway.New(modelgateway.Config{ProviderID: secure.ID(), Endpoint: openrouter.Endpoint, AllowedEndpoints: []string{openrouter.Endpoint}, Model: model, APIKey: key, FreeOnly: true})
	if err != nil {
		return err
	}
	defer gateway.Close()
	probe := gateway.Probe(ctx, nil)
	if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"model_id": model, "policy": "free_only", "probe": probe}); err != nil {
		return err
	}
	if !probe.Compatible() {
		return problem.New(502, probe.ErrorCode, "Free model protocol check failed")
	}
	return nil
}
