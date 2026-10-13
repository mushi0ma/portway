package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"portway/internal/codexbridge"
	"portway/internal/config"
)

// cliBinary resolves an optional CLI: explicit absolute path or PATH lookup; "" = not installed.
func cliBinary(variable, name string) (string, error) {
	if v := os.Getenv(variable); v != "" {
		if !filepath.IsAbs(v) {
			return "", errors.New(variable + " must be an absolute path")
		}
		if path, err := exec.LookPath(v); err == nil {
			return path, nil
		}
		return "", nil
	}
	path, _ := exec.LookPath(name)
	return path, nil
}
func codexCommand() error {
	key, err := config.Secret("PORTWAY_CODEX_BRIDGE_KEY", true)
	if err != nil {
		return err
	}
	workspace := os.Getenv("PORTWAY_CODEX_WORKSPACE")
	if !filepath.IsAbs(workspace) {
		return errors.New("PORTWAY_CODEX_WORKSPACE must be an absolute directory on this laptop")
	}
	workspace, err = filepath.EvalSymlinks(workspace)
	if err != nil {
		return errors.New("Codex workspace is unavailable")
	}
	st, err := os.Stat(workspace)
	if err != nil || !st.IsDir() {
		return errors.New("Codex workspace must be a directory")
	}
	codexBinary, err := cliBinary("PORTWAY_CODEX_BINARY", "codex")
	if err != nil {
		return err
	}
	claudeBinary, err := cliBinary("PORTWAY_CLAUDE_BINARY", "claude")
	if err != nil {
		return err
	}
	if codexBinary == "" && claudeBinary == "" {
		return errors.New("install Codex CLI or Claude Code on this laptop first")
	}
	root := os.Getenv("PORTWAY_BRIDGE_DATA_DIR")
	if root == "" {
		if root, err = codexbridge.DefaultRegistryRoot(); err != nil {
			return errors.New("bridge data directory is unavailable")
		}
	}
	accounts, err := codexbridge.OpenRegistry(root)
	if err != nil {
		return errors.New("bridge account registry is unavailable; check " + root + " permissions")
	}
	bridge := &codexbridge.Bridge{Workspace: workspace, Accounts: accounts, Agents: []codexbridge.Agent{
		&codexbridge.Codex{Binary: codexBinary, Workspace: workspace},
		&codexbridge.Claude{Binary: claudeBinary, Workspace: workspace},
	}}
	worker, err := codexbridge.NewWorker(strings.TrimRight(os.Getenv("PORTWAY_CODEX_API_URL"), "/"), key, bridge)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	fmt.Printf("Starting Portway agent bridge (codex=%t, claude=%t, workspace=portway, sandbox=read-only; credentials omitted)\n", codexBinary != "", claudeBinary != "")
	return worker.Serve(ctx)
}
