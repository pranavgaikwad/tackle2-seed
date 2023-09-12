package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/konveyor/tackle2-seed/pkg"
	"github.com/pborman/getopt/v2"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	Resources      = "resources"
	RuleSets       = "rulesets"
	RemoteRuleSets = "default/generated"
)

var Deps = []string{
	"rulesets/00-discovery",
	"rulesets/technology-usage",
}

var YesAssumed = true

func main() {
	cmd := Cmd{}
	err := cmd.Main()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

type Cmd struct {
	Path     string
	Remote   Remote
	Manifest struct {
		Current Manifest
		Remote  Manifest
	}
}

func (r *Cmd) Main() (err error) {
	resources := getopt.StringLong(
		"path",
		'p',
		"./"+Resources,
		"The resources path.")
	remote := getopt.StringLong(
		"remote",
		'r',
		"https://github.com/konveyor/rulesets",
		"The remote (ruleset) github repository URL. May be plan file path.")
	ref := getopt.StringLong(
		"branch",
		'b',
		"",
		"The github branch (any ref).")
	yes := getopt.BoolLong(
		"yes",
		'y',
		"Yes assumed.")
	help := getopt.BoolLong(
		"help",
		'h',
		"Show help.")

	getopt.Parse()
	if *help {
		getopt.Usage()
		return
	}

	YesAssumed = *yes
	r.Path = *resources
	r.Remote.URL = *remote
	r.Remote.Ref = *ref
	r.Manifest.Current.Root = *resources

	fmt.Printf("\nResources: %s\n", *resources)
	fmt.Printf("Remote: %s\n", *remote)
	fmt.Printf("Ref: %s\n", *ref)

	err = r.Manifest.Current.Load()
	if err != nil {
		return
	}
	err = r.Remote.Load()
	if err != nil {
		return
	}
	err = r.Reconcile()
	if err != nil {
		return
	}

	fmt.Println("\nDone")
	return
}

func (r *Cmd) Reconcile() (err error) {
	bash := Bash{}
	tmpDir, err := ioutil.TempDir("", "ruleset-*")
	if err != nil {
		return
	}
	remote := path.Join(r.Remote.Path, RemoteRuleSets)
	dest := path.Join(tmpDir, RuleSets)
	err = bash.Run("cp", "-r", remote, dest)
	if err != nil {
		return
	}
	r.Manifest.Remote.Root = tmpDir
	err = r.Manifest.Remote.Build()
	if err != nil {
		return
	}
	r.Manifest.Current.Reconcile(r.Manifest.Remote)
	r.Manifest.Current.PrintChanged()
	if !r.Manifest.Current.Dirty() {
		return
	}
	b := Ask("Apply approved changes?")
	if b {
		err = r.Apply()
		if err != nil {
			return
		}
	}
	return
}

func (r *Cmd) Apply() (err error) {
	manifest := r.Manifest.Current
	for _, ruleSet := range manifest.changed.added {
		err = r.ReplaceDir(ruleSet)
		if err != nil {
			return
		}
	}
	for _, ruleSet := range manifest.changed.updated {
		err = r.ReplaceDir(ruleSet)
		if err != nil {
			return
		}
	}
	for _, ruleSet := range manifest.changed.deleted {
		err = r.Delete(ruleSet)
		if err != nil {
			return
		}
	}
	err = r.Manifest.Current.Write()
	if err != nil {
		return
	}
	bash := Bash{}
	bash.Dir = r.Path
	err = bash.Run("git add", RuleSets)
	if err != nil {
		return
	}
	return
}

func (r *Cmd) ReplaceDir(ruleSet *pkg.RuleSet) (err error) {
	bash := Bash{Silent: true}
	remote := path.Join(r.Manifest.Remote.Root, ruleSet.Dir())
	current := path.Join(r.Path, ruleSet.Dir())
	err = bash.Run("rm -rf", current)
	if err != nil {
		return
	}
	bash = Bash{}
	err = bash.Run("cp -r", remote, current)
	return
}
func (r *Cmd) Delete(ruleSet *pkg.RuleSet) (err error) {
	bash := Bash{}
	p := path.Join(r.Path, ruleSet.Dir())
	err = bash.Run("rm -rf", p)
	return
}

type Remote struct {
	URL  string
	Ref  string
	Path string
}

func (r *Remote) Load() (err error) {
	if strings.HasPrefix(r.URL, "/") {
		r.Path = r.URL
		return
	}
	r.Path, err = ioutil.TempDir("", "ruleset-*")
	if err != nil {
		return
	}
	bash := Bash{}
	err = bash.Run("git clone", r.URL, r.Path)
	if err != nil {
		return
	}
	if r.Ref != "" {
		bash.Dir = r.Path
		err = bash.Run("git checkout", r.Ref)
		if err != nil {
			return
		}
	}
	return
}

type Manifest struct {
	Root     string
	pkg.Seed `yaml:",inline"`
	ruleSets []*pkg.RuleSet
	dirMap   map[string]*pkg.RuleSet
	changed  struct {
		skipped map[string]*pkg.RuleSet
		added   []*pkg.RuleSet
		updated []*pkg.RuleSet
		deleted []*pkg.RuleSet
	}
}

func (r *Manifest) Load() (err error) {
	r.dirMap = make(map[string]*pkg.RuleSet)
	p := path.Join(r.Root, "rulesets.yaml")
	fmt.Printf("[Manifest] Read: %s\n", p)
	f, err := os.Open(p)
	if err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()
	d := yaml.NewDecoder(f)
	err = d.Decode(r)
	if err != nil {
		return
	}
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
	p := path.Join(r.Root, "rulesets.yaml")
	fmt.Printf("[Manifest] Write: %s\n", p)
	r.Items = []yaml.Node{}
	for _, ruleSet := range r.ruleSets {
		node := yaml.Node{}
		err = node.Encode(ruleSet)
		if err != nil {
			return
		}
		r.Items = append(r.Items, node)
	}
	f, err := os.Create(p)
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
		err = r.SetName(ruleSet)
		if err != nil {
			return
		}
		err = r.SetDigest(ruleSet)
		if err != nil {
			return
		}
		r.SetDeps(ruleSet)
		r.ruleSets = append(r.ruleSets, ruleSet)
		r.dirMap[ruleSet.Dir()] = ruleSet
	}
	return
}

func (r *Manifest) Reconcile(other Manifest) {
	r.changed.skipped = make(map[string]*pkg.RuleSet)
	for _, ruleSet := range other.ruleSets {
		if r.IsDep(ruleSet) {
			r.changed.skipped[ruleSet.Dir()] = ruleSet
			continue
		}
		matched, found := r.dirMap[ruleSet.Dir()]
		if !found {
			b := Ask("RuleSet at: %s unknown. Add?", ruleSet.Dir())
			if b {
				r.Add(ruleSet)
				r.changed.added = append(
					r.changed.added,
					ruleSet)
			}
		} else {
			if matched.Checksum != ruleSet.Checksum {
				b := Ask("RuleSet at: %s changed. Update?", ruleSet.Dir())
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
		if r.IsDep(ruleSet) {
			r.changed.skipped[ruleSet.Dir()] = ruleSet
			continue
		}
		if _, found := other.dirMap[ruleSet.Dir()]; !found {
			b := Ask(
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

func (r *Manifest) SetName(ruleSet *pkg.RuleSet) (err error) {
	p := path.Join(r.Root, ruleSet.Dir(), "ruleset.yaml")
	object := struct {
		Name string
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
		"\n[Manifest] summary: (S)kipped=%d,(A)dded=%d,(M)odified=%d,(D)eleted=%d\n",
		len(r.changed.skipped),
		len(r.changed.added),
		len(r.changed.updated),
		len(r.changed.deleted))
	for _, ruleSet := range r.changed.skipped {
		fmt.Printf("  S (%s) %s\n", ruleSet.Name, ruleSet.Dir())
	}
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

type Bash struct {
	Dir    string
	Silent bool
}

func (r *Bash) Run(args ...string) (err error) {
	command := strings.Join(args, " ")
	if !r.Silent {
		fmt.Printf("[CMD] %s\n", command)
	}
	cmd := exec.Command("bash", "--norc", "-c", command)
	cmd.Dir = r.Dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	if !r.Silent {
		fmt.Print(string(output))
	}
	return
}

func Ask(prompt string, v ...any) (b bool) {
	if YesAssumed {
		b = true
		return
	}
	fmt.Printf(prompt, v...)
	for {
		fmt.Print(" [Y|n]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if answer != "" {
			switch answer[0] {
			case '\n', 'Y', 'y':
				b = true
				return
			case 'N', 'n':
				return
			}
		}
	}
}
