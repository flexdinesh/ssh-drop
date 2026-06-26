package session

import "errors"

var ErrTransferCanceled = errors.New("transfer canceled")

type Config struct {
	Remotes []Remote
}

func (c Config) FindRemote(name string) *Remote {
	for i := range c.Remotes {
		if c.Remotes[i].Name == name {
			return &c.Remotes[i]
		}
	}
	return nil
}

type Remote struct {
	Name         string
	Host         string
	User         string
	Port         string
	IdentityFile string
	ForwardAgent bool
	Destination  string
}

func (r Remote) Target() string {
	if r.User == "" {
		return r.Host
	}
	return r.User + "@" + r.Host
}

type Start struct {
	Config            Config
	PreselectedRemote string
}

type Summary struct {
	Successes              int
	Failures               int
	Canceled               int
	SuccessfulDestinations []string
}

func (s Summary) Empty() bool {
	return s.Successes == 0 && s.Failures == 0 && s.Canceled == 0 && len(s.SuccessfulDestinations) == 0
}

type TransferRequest struct {
	LocalPath       string
	DestinationDir  string
	DestinationPath string
	Remote          Remote
	Password        string
}

type TransferEvent struct {
	Output string
	Done   bool
	Err    error
}
