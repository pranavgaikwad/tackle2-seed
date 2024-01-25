package pkg

import (
	"os"
	"path"
	"strings"

	liberr "github.com/jortel/go-utils/error"
	"gopkg.in/yaml.v3"
)

// RuleSet constants
const (
	RuleSuffix   = ".yaml"
	RuleSetYaml  = "ruleset.yaml"
	RuleSetImage = "image.svg"
)

// Rule seed representation.
type Rule struct {
	labels   map[string]bool
	checksum []byte
	Path     string
}

// AppendLabel adds a label to the rule without duplication.
func (r *Rule) AppendLabel(label string) {
	if r.labels == nil {
		r.labels = make(map[string]bool)
	}
	r.labels[label] = true
}

// Labels returns a slice of the rule's labels.
func (r *Rule) Labels() (labels []string) {
	for l, _ := range r.labels {
		labels = append(labels, l)
	}
	return
}

// RuleSet seed representation.
type RuleSet struct {
	UUID         string   `yaml:",omitempty"`
	Name         string   `yaml:",omitempty"`
	Description  string   `yaml:",omitempty"`
	Labels       []string `yaml:",omitempty"`
	Directory    string   `yaml:",omitempty"`
	Dependencies []string `yaml:",omitempty"`
	Checksum     string   `yaml:",omitempty"`
	Rules        []Rule   `yaml:"-"`
	SeedDir      string   `yaml:"-"`
}

// Dir returns the path to the directory containing the rule files.
func (r *RuleSet) Dir() string {
	return path.Join(r.SeedDir, r.Directory)
}

// Yaml returns the path to the ruleset.yaml file.
func (r *RuleSet) Yaml() string {
	return path.Join(r.SeedDir, r.Directory, RuleSetYaml)
}

// Load populates the seed representation with values
// from the analyzer ruleset yaml.
func (r *RuleSet) Load() (err error) {
	err = r.readRuleSet()
	if err != nil {
		return
	}
	err = r.readRules()
	if err != nil {
		return
	}
	return
}

// readRuleSet reads the analyzer ruleset yaml to set
// fields on the seed RuleSet representation.
func (r *RuleSet) readRuleSet() (err error) {
	type analyzerRuleset struct {
		Name        string
		Description string
		Labels      []string
	}
	rs := analyzerRuleset{}

	f, err := os.Open(r.Yaml())
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&rs)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	if r.Name == "" {
		r.Name = rs.Name
	}
	if r.Description == "" {
		r.Description = rs.Description
	}
	r.Labels = rs.Labels
	r.Rules = []Rule{{Path: r.Yaml()}}
	return
}

// readRules reads the analyzer rule files from the dir
// specified by the RuleSet's Directory field relative to the
// supplied base path, and uses the contents to populate
// seed Rule representations on the RuleSet.
func (r *RuleSet) readRules() (err error) {
	entries, err := os.ReadDir(r.Dir())
	if err != nil {
		err = liberr.Wrap(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == RuleSetYaml {
			continue
		}
		if !strings.HasSuffix(entry.Name(), RuleSuffix) {
			continue
		}
		err = func() (err error) {
			filePath := path.Join(r.Dir(), entry.Name())
			f, err := os.Open(filePath)
			if err != nil {
				err = liberr.Wrap(err)
				return
			}
			defer f.Close()

			type analyzerRule struct {
				Labels []string
			}
			analyzerRules := []analyzerRule{}
			decoder := yaml.NewDecoder(f)
			err = decoder.Decode(&analyzerRules)
			if err != nil {
				err = liberr.Wrap(err)
				return
			}

			rule := Rule{Path: filePath}
			for _, ar := range analyzerRules {
				for _, label := range ar.Labels {
					rule.AppendLabel(label)
				}
			}
			for _, label := range r.Labels {
				rule.AppendLabel(label)
			}
			r.Rules = append(r.Rules, rule)
			return
		}()
	}

	return
}
