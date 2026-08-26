package steps

import "ilmango/internal/installer"

// base carries the identity fields every step shares, so a concrete step is
// just its Run method plus a descriptor.
type base struct {
	id      string
	title   string
	detail  string
	applies func(installer.Config) bool
}

func (b base) ID() string          { return b.id }
func (b base) Title() string       { return b.title }
func (b base) Description() string { return b.detail }

func (b base) AppliesTo(cfg installer.Config) bool {
	if b.applies == nil {
		return true
	}
	return b.applies(cfg)
}
