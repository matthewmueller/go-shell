package shell

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
)

// Commands is a command executor
type Commands interface {
	Command(name string, args ...string) *Cmd
}

// Command is a single command that can be started or run
type Command interface {
	Start() (*Process, error)
	Run(ctx context.Context) (string, error)
	Execute(ctx context.Context) error
}

func Default() *Exec {
	return New(".")
}

func New(dir string) *Exec {
	return &Exec{
		Dir:    dir,
		Env:    os.Environ(),
		Stderr: os.Stderr,
		Stdout: os.Stdout,
		Stdin:  os.Stdin,
	}
}

type Exec struct {
	Dir    string
	Env    []string
	Stderr io.Writer
	Stdout io.Writer
	Stdin  io.Reader
}

var _ Commands = (*Exec)(nil)

func (c *Exec) Command(name string, args ...string) *Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = c.Dir
	cmd.Env = c.Env
	cmd.Stderr = c.Stderr
	cmd.Stdout = c.Stdout
	cmd.Stdin = c.Stdin
	return (*Cmd)(cmd)
}

type Cmd exec.Cmd

var _ Command = (*Cmd)(nil)

func (c *Cmd) exec() *exec.Cmd {
	return (*exec.Cmd)(c)
}

// Start the command, returning a process.
func (c *Cmd) Start() (*Process, error) {
	if err := c.exec().Start(); err != nil {
		return nil, err
	}
	p := newProcess(c)
	go p.wait()
	return p, nil
}

// Run the command returning stdout
func (c *Cmd) Run(ctx context.Context) (stdout string, err error) {
	out := new(bytes.Buffer)
	c.Stdout = out
	p, err := c.Start()
	if err != nil {
		return "", err
	}
	if err := p.Wait(ctx); err != nil {
		return "", err
	}
	return out.String(), nil
}

// Execute the command
func (c *Cmd) Execute(ctx context.Context) (err error) {
	p, err := c.Start()
	if err != nil {
		return err
	}
	if err := p.Wait(ctx); err != nil {
		return err
	}
	return nil
}
