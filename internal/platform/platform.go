package platform

import "github.com/charlesonunze/nuke/internal/nuke"

func New() (nuke.System, error) {
	return newSystem()
}
