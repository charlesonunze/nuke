//go:build windows

package platform

import (
	"context"
	"errors"

	"github.com/charlesonunze/nuke/internal/nuke"
)

var errWindowsUnsupported = errors.New("windows support is not implemented yet")

type system struct{}

func newSystem() (nuke.System, error) {
	return system{}, nil
}

func (system) FindPortOwners(context.Context, nuke.PortTarget) ([]nuke.PortOwner, error) {
	return nil, errWindowsUnsupported
}

func (system) FindProcesses(context.Context, nuke.ProcessTarget) ([]nuke.Process, error) {
	return nil, errWindowsUnsupported
}

func (system) Terminate(context.Context, int, nuke.Signal) error {
	return errWindowsUnsupported
}
