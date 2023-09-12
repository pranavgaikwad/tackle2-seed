package main

import (
	"fmt"
	"github.com/konveyor/tackle2-seed/pkg"
	"github.com/pborman/getopt/v2"
	"io/ioutil"
	"os"
	"path"
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
	b := bash.Ask("Apply approved changes?")
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
