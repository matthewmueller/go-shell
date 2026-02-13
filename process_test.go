package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/matthewmueller/testchild"
)

func TestCmdRun(t *testing.T) {
	is := is.New(t)
	cmd := New("")
	is.NoErr(shellCommand(t, cmd, "exit 0").Run(context.Background()))
}

func TestCmdRunError(t *testing.T) {
	is := is.New(t)
	cmd := New("")
	err := shellCommand(t, cmd, "exit 7").Run(context.Background())
	is.Equal(err == nil, false)
}

func TestProcessStopAfterExitDoesNotLeakGoroutines(t *testing.T) {
	is := is.New(t)
	base := runtime.NumGoroutine()
	cmd := New("")

	for i := 0; i < 120; i++ {
		p, err := shellCommand(t, cmd, "exit 0").Start()
		is.NoErr(err)
		time.Sleep(3 * time.Millisecond)
		is.NoErr(p.Stop(context.Background()))
	}

	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	after := runtime.NumGoroutine()
	delta := after - base
	is.Equal(delta > 20, false)
}

func TestProcessRestart(t *testing.T) {
	is := is.New(t)
	out := new(bytes.Buffer)
	cmd := New("")
	cmd.Stdout = out
	cmd.Stderr = io.Discard

	p, err := shellCommand(t, cmd, "echo restart-ok").Start()
	is.NoErr(err)
	is.NoErr(p.Wait(context.Background()))

	next, err := p.Restart(context.Background())
	is.NoErr(err)
	is.NoErr(next.Wait(context.Background()))

	is.Equal(strings.Count(out.String(), "restart-ok"), 2)
}

func TestProcessWaitContextCanceledKillsProcess(t *testing.T) {
	is := is.New(t)
	cmd := New("")
	p, err := sleepCommand(t, cmd, 5).Start()
	is.NoErr(err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	is.NoErr(p.Wait(ctx))
	is.Equal(time.Since(start) < 2*time.Second, true)
}

func TestProcessStopAfterWaitReturnsNil(t *testing.T) {
	is := is.New(t)
	cmd := New("")
	p, err := shellCommand(t, cmd, "exit 0").Start()
	is.NoErr(err)

	is.NoErr(p.Wait(context.Background()))
	is.NoErr(p.Stop(context.Background()))
}

func TestProcessRestartPreservesDirAndEnv(t *testing.T) {
	is := is.New(t)
	token := "restart-token-123"
	dir := t.TempDir()
	out := new(bytes.Buffer)
	cmd := New(dir)
	cmd.Stdout = out
	cmd.Stderr = io.Discard
	cmd.Env = append(os.Environ(), "GO_SHELL_RESTART_TOKEN="+token)

	p, err := restartProbeCommand(t, cmd).Start()
	is.NoErr(err)
	is.NoErr(p.Wait(context.Background()))

	next, err := p.Restart(context.Background())
	is.NoErr(err)
	is.NoErr(next.Wait(context.Background()))

	expectDir, err := filepath.EvalSymlinks(dir)
	is.NoErr(err)
	expect := token + "|" + expectDir
	actual := strings.ReplaceAll(out.String(), "\r\n", "\n")
	is.Equal(strings.Count(actual, expect), 2)
}

func TestProcessStopContextCancelFallsBackToKill(t *testing.T) {
	testchild.Run(t, func(t testing.TB, child *exec.Cmd) {
		is := is.New(t)
		child.Stdout = io.Discard
		child.Stderr = io.Discard

		p, err := ((*Cmd)(child)).Start()
		is.NoErr(err)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		is.NoErr(p.Stop(ctx))
		is.Equal(time.Since(start) < 2*time.Second, true)
	}, func(t testing.TB) {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt)
		defer signal.Stop(sigs)

		deadline := time.NewTimer(10 * time.Second)
		defer deadline.Stop()
		for {
			select {
			case <-sigs:
				// Ignore interrupt so parent must cancel context and kill.
			case <-deadline.C:
				return
			}
		}
	})
}

func shellCommand(t *testing.T, cmd *Exec, script string) *Cmd {
	t.Helper()
	if runtime.GOOS == "windows" {
		return cmd.Command("cmd", "/C", script)
	}
	return cmd.Command("sh", "-c", script)
}

func restartProbeCommand(t *testing.T, cmd *Exec) *Cmd {
	t.Helper()
	if runtime.GOOS == "windows" {
		return cmd.Command("cmd", "/C", "echo %GO_SHELL_RESTART_TOKEN%^|%CD%")
	}
	return cmd.Command("sh", "-c", `printf "%s|%s\n" "$GO_SHELL_RESTART_TOKEN" "$(pwd -P)"`)
}

func sleepCommand(t *testing.T, cmd *Exec, seconds int) *Cmd {
	t.Helper()
	if runtime.GOOS == "windows" {
		return cmd.Command("cmd", "/C", "ping -n 6 127.0.0.1 >NUL")
	}
	return cmd.Command("sh", "-c", fmt.Sprintf("sleep %d", seconds))
}
