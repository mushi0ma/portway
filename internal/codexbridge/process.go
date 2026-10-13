package codexbridge

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Explicit environment prevents inheriting API, database, bridge or model keys.
// Only the OS basics and the selected account profile directory reach a CLI.
var baseKeys = []string{"HOME", "PATH", "USER", "SHELL", "SSL_CERT_FILE", "SSL_CERT_DIR", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"}

// Display variables are passed only to a "computer" login so the laptop browser can open.
var displayKeys = []string{"DISPLAY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS", "XAUTHORITY", "XDG_CURRENT_DESKTOP", "XDG_SESSION_TYPE"}

func nativeEnv(extra ...string) []string {
	out := []string{"LANG=C.UTF-8"}
	for _, key := range baseKeys {
		if v := os.Getenv(key); v != "" {
			out = append(out, key+"="+v)
		}
	}
	return append(out, extra...)
}
func displayEnv() []string {
	out := []string{}
	for _, key := range displayKeys {
		if v := os.Getenv(key); v != "" {
			out = append(out, key+"="+v)
		}
	}
	return out
}

// native starts one CLI process in its own process group; cancellation kills the group.
func native(ctx context.Context, binary, dir string, env []string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = 2 * time.Second
	return cmd
}
