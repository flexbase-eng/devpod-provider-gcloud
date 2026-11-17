package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/loft-sh/devpod-provider-gcloud/pkg/gcloud"
	"github.com/loft-sh/devpod-provider-gcloud/pkg/options"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/ssh"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// CommandCmd holds the cmd flags
type CommandCmd struct{}

// NewCommandCmd defines a command
func NewCommandCmd() *cobra.Command {
	cmd := &CommandCmd{}
	commandCmd := &cobra.Command{
		Use:   "command",
		Short: "Run a command on the instance",
		RunE: func(_ *cobra.Command, args []string) error {
			options, err := options.FromEnv(true, true)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), options, log.Default)
		},
	}

	return commandCmd
}

// Run runs the command logic
func (cmd *CommandCmd) Run(ctx context.Context, options *options.Options, log log.Logger) error {
	command := os.Getenv("COMMAND")
	if command == "" {
		return fmt.Errorf("command environment variable is missing")
	}

	// get private key
	privateKey, err := ssh.GetPrivateKeyRawBase(options.MachineFolder)
	if err != nil {
		return fmt.Errorf("load private key: %w", err)
	}

	// create gcloud client
	client, err := gcloud.NewClient(ctx, options.Project, options.Zone)
	if err != nil {
		return err
	}
	defer client.Close()

	// get instance
	instance, err := client.Get(ctx, options.MachineID)
	if err != nil {
		return err
	} else if instance == nil {
		return fmt.Errorf("instance %s doesn't exist", options.MachineID)
	}

	// get external ip
	if options.PublicIP && (len(instance.NetworkInterfaces) == 0 || len(instance.NetworkInterfaces[0].AccessConfigs) == 0 || instance.NetworkInterfaces[0].AccessConfigs[0].NatIP == nil) {
		return fmt.Errorf("instance %s doesn't have an external nat ip", options.MachineID)
	}

	if options.PublicIP {
		sshClient, err := ssh.NewSSHClient("devpod", *instance.NetworkInterfaces[0].AccessConfigs[0].NatIP+":22", privateKey)
		if err != nil {
			return errors.Wrap(err, "create ssh client")
		}
		defer sshClient.Close()

		return ssh.Run(ctx, sshClient, command, os.Stdin, os.Stdout, os.Stderr)
	}

	// The tunnel carries SSH over its own stdin/stdout rather than a local port, so it exits when that
	// pipe closes. devpod kills this process without letting deferred cleanup run, and a tunnel bound
	// to a local port would outlive it.
	tunnel := exec.Command("gcloud", []string{
		"compute",
		"start-iap-tunnel",
		*instance.Name,
		"22",
		"--listen-on-stdin",
		"--zone=" + options.Zone,
	}...)
	tunnelStderr := &tailBuffer{max: 4096}
	tunnel.Stderr = tunnelStderr
	tunnelIn, err := tunnel.StdinPipe()
	if err != nil {
		return fmt.Errorf("start tunnel: %w", err)
	}
	tunnelOut, err := tunnel.StdoutPipe()
	if err != nil {
		return fmt.Errorf("start tunnel: %w", err)
	}
	if err = tunnel.Start(); err != nil {
		return fmt.Errorf("start tunnel: %w", err)
	}
	defer func() {
		_ = tunnelIn.Close()
		_ = tunnel.Process.Kill()
		_ = tunnel.Wait()
	}()

	sshClient, err := ssh.StdioClientFromKeyBytesWithUser(privateKey, tunnelOut, tunnelIn, "devpod", false)
	if err != nil {
		return errors.Wrapf(err, "create ssh client over iap tunnel: %s", tunnelStderr.String())
	}
	defer sshClient.Close()

	return ssh.Run(ctx, sshClient, command, os.Stdin, os.Stdout, os.Stderr)
}

// tailBuffer keeps the last max bytes written to it, so a long session cannot grow it without bound.
type tailBuffer struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		b.buf = b.buf[len(b.buf)-b.max:]
	}
	return len(p), nil
}

func (b *tailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.TrimSpace(string(b.buf))
}
