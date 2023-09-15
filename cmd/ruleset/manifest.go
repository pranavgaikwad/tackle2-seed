package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/konveyor/tackle2-seed/pkg"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type Manifest struct {
	pkg.Seed `yaml:",inline"`
	Root     string
	path     string
	ruleSets []*pkg.RuleSet
	dirMap   map[string]*pkg.RuleSet
	changed  struct {
		added   []*pkg.RuleSet
		updated []*pkg.RuleSet
		deleted []*pkg.RuleSet
	}
}

func (r *Manifest) Load() (err error) {
	r.dirMap = make(map[string]*pkg.RuleSet)
	err = r.find()
	if err != nil {
		return
	}
	fmt.Printf("[Manifest] Read: %s\n", r.path)
	for _, node := range r.Items {
		ruleSet := &pkg.RuleSet{}
		err = node.Decode(ruleSet)
		if err == nil {
			r.ruleSets = append(r.ruleSets, ruleSet)
			r.dirMap[ruleSet.Dir()] = ruleSet
		} else {
			return
		}
	}
	return
}
func (r *Manifest) Write() (err error) {
	fmt.Printf("[Manifest] Write: %s\n", r.path)
	r.Items = []yaml.Node{}
	for _, ruleSet := range r.ruleSets {
		node := yaml.Node{}
		err = node.Encode(ruleSet)
		if err != nil {
			return
		}
		r.Items = append(r.Items, node)
	}
	f, err := os.Create(r.path)
	if err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()
	encoder := yaml.NewEncoder(f)
	err = encoder.Encode(r)
	return
}

func (r *Manifest) Build() (err error) {
	r.dirMap = make(map[string]*pkg.RuleSet)
	p := path.Join(r.Root, RuleSets)
	entries, err := os.ReadDir(p)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		ruleSet := &pkg.RuleSet{
			Directory: path.Join(RuleSets, entry.Name()),
		}
		err = r.SetDetails(ruleSet)
		if err != nil {
			return
		}
		r.ruleSets = append(r.ruleSets, ruleSet)
		r.dirMap[ruleSet.Dir()] = ruleSet
	}
	return
}

func (r *Manifest) Reconcile(other Manifest) {
	bash := Bash{}
	for _, ruleSet := range other.ruleSets {
		matched, found := r.dirMap[ruleSet.Dir()]
		if !found {
			b := bash.Ask("RuleSet at: %s unknown. Add?", ruleSet.Dir())
			if b {
				r.Add(ruleSet)
				r.changed.added = append(
					r.changed.added,
					ruleSet)
			}
		} else {
			if matched.Checksum != ruleSet.Checksum {
				b := bash.Ask("RuleSet at: %s changed. Update?", ruleSet.Dir())
				if b {
					r.Update(ruleSet)
					r.changed.updated = append(
						r.changed.updated,
						ruleSet)
				}
			}
		}
	}
	for _, ruleSet := range r.ruleSets {
		if _, found := other.dirMap[ruleSet.Dir()]; !found {
			b := bash.Ask(
				"RuleSet at: %s not-found. Delete?",
				ruleSet.Dir())
			if b {
				r.Delete(ruleSet)
				r.changed.deleted = append(
					r.changed.deleted,
					ruleSet)
			}
		}
	}
}

func (r *Manifest) Add(ruleSet *pkg.RuleSet) {
	r.ruleSets = append(
		r.ruleSets,
		ruleSet)
}

func (r *Manifest) Update(ruleSet *pkg.RuleSet) {
	for i := range r.ruleSets {
		if r.ruleSets[i].Dir() == ruleSet.Dir() {
			r.ruleSets[i].Description = ruleSet.Description
			if r.IsDep(ruleSet) {
				continue
			}
			r.ruleSets[i].Name = ruleSet.Name
			break
		}
	}
}

func (r *Manifest) Delete(ruleSet *pkg.RuleSet) {
	var wanted []*pkg.RuleSet
	for i := range r.ruleSets {
		if r.ruleSets[i].Dir() != ruleSet.Dir() {
			wanted = append(
				wanted,
				r.ruleSets[i])
		}
	}
	r.ruleSets = wanted
}

func (r *Manifest) SetDetails(ruleSet *pkg.RuleSet) (err error) {
	p := path.Join(r.Root, ruleSet.Dir(), "ruleset.yaml")
	object := struct {
		Name        string
		Description string
	}{}
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			ruleSet.Name = path.Base(ruleSet.Dir())
			err = nil
		}
		return
	}
	defer func() {
		_ = f.Close()
	}()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(b, &object)
	if err != nil {
		return
	}
	ruleSet.Name = object.Name
	ruleSet.Description = object.Description
	r.SetDeps(ruleSet)
	err = r.SetDigest(ruleSet)
	return
}

func (r *Manifest) SetDigest(ruleSet *pkg.RuleSet) (err error) {
	p := path.Join(r.Root, ruleSet.Dir())
	b, err := pkg.ChecksumDir(p)
	if err == nil {
		ruleSet.Checksum = hex.EncodeToString(b)
	}
	return
}

func (r *Manifest) SetDeps(ruleSet *pkg.RuleSet) {
	for _, d := range Deps {
		ruleSet.Dependencies = append(
			ruleSet.Dependencies,
			"@"+d)
	}
}

func (r *Manifest) IsDep(ruleSet *pkg.RuleSet) (b bool) {
	for _, d := range Deps {
		if d == ruleSet.Dir() {
			b = true
			break
		}
	}
	return
}

func (r *Manifest) PrintChanged() {
	fmt.Printf(
		"\n[Manifest] summary: (A)dded=%d,(M)odified=%d,(D)eleted=%d\n",
		len(r.changed.added),
		len(r.changed.updated),
		len(r.changed.deleted))
	for _, ruleSet := range r.changed.added {
		fmt.Printf("  A (%s) %s\n", ruleSet.Name, ruleSet.Dir())
	}
	for _, ruleSet := range r.changed.updated {
		fmt.Printf("  M (%s) %s\n", ruleSet.Name, ruleSet.Dir())
	}
	for _, ruleSet := range r.changed.deleted {
		fmt.Printf("  D (%s) %s\n", ruleSet.Name, ruleSet.Dir())
	}
	fmt.Println("")
}

func (r *Manifest) Dirty() (b bool) {
	n := 0
	n += len(r.changed.added)
	n += len(r.changed.updated)
	n += len(r.changed.deleted)
	b = n > 0
	return
}

func (r *Manifest) find() (err error) {
	entries, err := os.ReadDir(r.Root)
	if err != nil {
		return
	}
	var f *os.File
	defer func() {
		_ = f.Close()
	}()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		r.path = path.Join(r.Root, entry.Name())
		f, err = os.Open(r.path)
		if err != nil {
			return
		}
		d := yaml.NewDecoder(f)
		err = d.Decode(r)
		_ = f.Close()
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			return
		}
		if strings.ToLower(r.Kind) == pkg.KindRuleSet {
			return
		}
	}
	err = fmt.Errorf("manifest not found")
	return
}
