package main

import (
	"io/ioutil"
	"strings"
)

type Remote struct {
	URL  string
	Ref  string
	Path string
}

func (r *Remote) Load() (err error) {
	if strings.HasPrefix(r.URL, "/") || strings.HasPrefix(r.URL, ".") {
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
