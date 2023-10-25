package pkg

import "path"

type Target struct {
	UUID        string        `yaml:",omitempty"`
	Name        string        `yaml:",omitempty"`
	Description string        `yaml:",omitempty"`
	Provider    string        `yaml:"provider"`
	ImagePath   string        `yaml:",omitempty"`
	Choice      bool          `yaml:",omitempty"`
	Labels      []TargetLabel `yaml:",omitempty"`
	SeedDir     string        `yaml:",omitempty"`
}

func (r *Target) Image() string {
	return path.Join(r.SeedDir, r.ImagePath)
}

type TargetLabel struct {
	Name  string `yaml:",omitempty" json:"name"`
	Label string `yaml:",omitempty" json:"label"`
}
