package ssh

import (
	"errors"
	"os"
	"os/exec"
)

// A Client is a remote location where commands can be run through SSH and
// directories/files copied via rsync.
type Client struct {
	opts SSHOptions
}

type SSHOptions struct {
	Host       string
	User       string
	KnownHosts *KnownHosts
}

// New constructs a new Client pointing at the given host and username. If
// KnownHosts is non-nil, the ssh command used will use it as a known_hosts
// file.
func New(opts SSHOptions) *Client {
	return &Client{opts: opts}
}

// RunCommand runs the given command as a shell command on the remote host.
func (c *Client) RunCommand(cmd string) error {
	args := c.sshArgs()
	args = append(args, c.opts.Host, cmd)
	return runPassthrough(exec.Command("ssh", args...))
}

// RsyncTo runs rsync to copy from the given local source to the given remote
// destination.
func (c *Client) RsyncTo(src, dst string) error {
	cmd := exec.Command(
		"rsync",
		"-avP",
		"-e", c.sshInvocation(),
		src,
		c.opts.Host+":"+dst)
	return runPassthrough(cmd)
}

// RsyncFrom runs rsync to copy from the given remote source to the given local
// destination.
func (c *Client) RsyncFrom(src, dst string) error {
	cmd := exec.Command(
		"rsync",
		"-avP",
		"-e", c.sshInvocation(),
		c.opts.Host+":"+src,
		dst)
	return runPassthrough(cmd)
}

func (c *Client) sshArgs() []string {
	args := []string{
		"-o", "PasswordAuthentication=no",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "User=" + c.opts.User,
	}

	if c.opts.KnownHosts != nil {
		args = append(args, "-o", "UserKnownHostsFile="+c.opts.KnownHosts.Filename)
	}

	return args
}

func (c *Client) sshInvocation() string {
	return "ssh " + ShellEscapeJoin(c.sshArgs())
}

func runPassthrough(cmd *exec.Cmd) error {
	if cmd.Stdout != nil || cmd.Stderr != nil {
		// TODO: log
		return errors.New("runPassthrough: command already has stdout or stderr set")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
