package ssh

import (
	"os"
	"os/exec"
)

// A Target is a remote location where commands can be run through SSH and
// directories/files copied via rsync.
type Target struct {
	Hostname   string
	Username   string
	KnownHosts *KnownHosts
}

// New constructs a new Target pointing at the given host and username. If
// KnownHosts is non-nil, the ssh command used will use it as a known_hosts
// file.
func New(host, user string, knownhosts *KnownHosts) *Target {
	return &Target{
		Hostname:   host,
		Username:   user,
		KnownHosts: knownhosts,
	}
}

// RunCommand runs the given command as a shell command on the remote host.
func (t *Target) RunCommand(cmd string) error {
	opts := t.sshOpts()
	opts = append(opts, t.Hostname, cmd)
	return runExecPassthrough(exec.Command("ssh", opts...))
}

// RsyncTo runs rsync to copy from the given local source to the given remote
// destination.
func (t *Target) RsyncTo(src, dst string) error {
	c := exec.Command("rsync", "-avP", "-e", t.sshInvocation(), src, t.Hostname+":"+dst)
	return runExecPassthrough(c)
}

// RsyncFrom runs rsync to copy from the given remote source to the given local
// destination.
func (t *Target) RsyncFrom(src, dst string) error {
	c := exec.Command("rsync", "-avP", "-e", t.sshInvocation(), t.Hostname+":"+src, dst)
	return runExecPassthrough(c)
}

func (t *Target) sshOpts() []string {
	args := []string{
		"-o", "PasswordAuthentication=no",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "User=" + t.Username,
	}

	if t.KnownHosts != nil {
		args = append(args, "-o", "UserKnownHostsFile="+t.KnownHosts.Filename)
	}

	return args
}

func (t *Target) sshInvocation() string {
	return "ssh " + ShellEscapeJoin(t.sshOpts())
}

func runExecPassthrough(c *exec.Cmd) error {
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
