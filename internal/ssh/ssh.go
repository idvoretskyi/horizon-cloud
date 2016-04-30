package ssh

import (
	"errors"
	"net"
	"os"
	"os/exec"
)

// A Client is a remote location where commands can be run through SSH and
// directories/files copied via rsync.
type Client struct {
	opts Options
}

// Options holds parameters for a Client.
type Options struct {
	// Host should be set to the hostname of the server to talk to. It can
	// optionally include a port number after a colon.
	Host string

	// User should be set to the username desired. If not set (equal to ""),
	// it defaults to the username of the currently running process.
	User string

	// KnownHosts will be passed to ssh as the known_hosts file. If nil, the
	// user's known_hosts file is used.
	//
	// It must not be closed while the Client is in use.
	KnownHosts *KnownHosts

	// IdentityFile will be passed as the -i option to ssh if set. If not set,
	// the user's identities will be used.
	IdentityFile string

	// Environment specifies extra environment variables to be passed to the
	// remote ssh session (via SendEnv.) If nil, no extra environment variables
	// are sent.
	Environment map[string]string
}

// New constructs a new Client pointing at the given host.
func New(opts Options) *Client {
	return &Client{opts: opts}
}

// RunInteractive runs an interactive shell.
func (c *Client) RunInteractive() error {
	return runPassthroughTwoWay(c.Command(""))
}

// RunCommand runs the given command as a shell command on the remote host.
//
// It passes ssh's stdout and stderr to the Go process's stdout and stderr.
func (c *Client) RunCommand(cmd string) error {
	return runPassthrough(c.Command(cmd))
}

// Command returns an *exec.Cmd that, when executed, will run ssh with the
// appropriate arguments to run the given shell command remotely.
//
// If the command given is the empty string, the exec.Cmd returned will not
// specify a command.
func (c *Client) Command(remoteCmd string) *exec.Cmd {
	args := c.sshArgs()
	args = append(args, c.targetHost())
	if remoteCmd != "" {
		args = append(args, remoteCmd)
	}
	cmd := exec.Command("ssh", args...)
	c.addEnvironment(cmd)
	return cmd
}

// RsyncTo runs rsync to copy from the given local source to the given remote
// destination.
//
// It passes rsync's stdout and stderr to the Go process's stdout and stderr.
func (c *Client) RsyncTo(src, dst string, linkDest string) error {
	args := []string{"-avz", "--delete"}
	if linkDest != "" {
		args = append(args, "--link-dest="+linkDest)
	}
	args = append(args, "-e", c.sshInvocation(), src, c.targetHost()+":"+dst)
	cmd := exec.Command("rsync", args...)
	c.addEnvironment(cmd)
	return runPassthrough(cmd)
}

// RsyncFrom runs rsync to copy from the given remote source to the given local
// destination.
//
// It passes rsync's stdout and stderr to the Go process's stdout and stderr.
func (c *Client) RsyncFrom(src, dst string) error {
	cmd := exec.Command(
		"rsync",
		"-avz",
		"--delete",
		"-e", c.sshInvocation(),
		c.targetHost()+":"+src,
		dst)
	c.addEnvironment(cmd)
	return runPassthrough(cmd)
}

func (c *Client) sshArgs() []string {
	args := []string{
		"-o", "PasswordAuthentication=no",
		"-o", "StrictHostKeyChecking=yes",
	}

	if c.opts.User != "" {
		args = append(args, "-o", "User="+c.opts.User)
	}
	if c.opts.KnownHosts != nil {
		args = append(args, "-o", "UserKnownHostsFile="+c.opts.KnownHosts.Filename)
	}
	if c.opts.IdentityFile != "" {
		args = append(args, "-i", c.opts.IdentityFile)
	}

	if c.opts.Environment != nil {
		for key := range c.opts.Environment {
			args = append(args, "-o", "SendEnv="+key)
		}
	}

	_, port, err := net.SplitHostPort(c.opts.Host)
	if err == nil {
		args = append(args, "-p", port)
	}

	return args
}

func (c *Client) targetHost() string {
	host, _, err := net.SplitHostPort(c.opts.Host)
	if err == nil {
		return host
	}
	return c.opts.Host
}

func (c *Client) addEnvironment(cmd *exec.Cmd) {
	env := os.Environ()
	if c.opts.Environment != nil {
		for key, value := range c.opts.Environment {
			env = append(env, key+"="+value)
		}
	}
	cmd.Env = env
}

func (c *Client) sshInvocation() string {
	return "ssh " + ShellEscapeJoin(c.sshArgs())
}

func runPassthroughTwoWay(cmd *exec.Cmd) error {
	if cmd.Stdin != nil {
		return errors.New("runPassthroughTwoWay: command already has stdin set")
	}
	cmd.Stdin = os.Stdin
	return runPassthrough(cmd)
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
