# go-shell

[![Go Reference](https://pkg.go.dev/badge/github.com/matthewmueller/go-shell.svg)](https://pkg.go.dev/github.com/matthewmueller/go-shell)

`go-shell` is a small Go package that wraps `os/exec` with a process API.

This allows you to centrally manage defaults like the environment and stdio.

## Features

- starting processes
- waiting with cancellation
- graceful stop with fallback kill
- restarting a command with preserved configuration

## Install

```sh
go get github.com/matthewmueller/go-shell
```

## Example

```go
package main

import (
	"context"
	"log"

	shell "github.com/matthewmueller/go-shell"
)

func main() {
	exec := shell.New("")
	if err := exec.Command("go", "version").Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

## API Reference

- `New(dir string) *Exec`: creates an executor with inherited stdio and environment.
- `(*Exec).Command(name string, args ...string) *Cmd`: builds a configured command.
- `(*Cmd).Run(ctx context.Context) error`: starts and waits for completion.
- `(*Cmd).Start() (Process, error)`: starts the command and returns a running `Process`.
- `Process.Wait(ctx) error`: waits for exit; if `ctx` is canceled, the process is killed.
- `Process.Stop(ctx) error`: interrupts first, then kills if interruption cannot complete.
- `Process.Restart(ctx) (Process, error)`: stops the process, then starts the same command with the same `Dir`, `Env`, stdio, and extra files.

## Development

```sh
go test ./...
```
