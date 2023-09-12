package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/konveyor/tackle2-seed/pkg"
	"github.com/pborman/getopt/v2"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"strings"
)

func main() {
	help := getopt.BoolLong(
		"help",
		'h',
		"Show help.")
	input := getopt.StringLong(
		"input",
		'i',
		"./resources",
		"The resources path (input).")
	output := getopt.StringLong(
		"output",
		'o',
		"./resources",
		"The resources path (output).")
	getopt.Parse()
	if *help {
		getopt.Usage()
		return
	}
	
	inPath := *input
	outPath := *output

	seeds, _, err := pkg.ReadFromDir(inPath, pkg.AllVersions)
	if err != nil {
		panic(err)
	}
	seedsByFile := make(map[string][]*pkg.Seed)
	rulesetUUIDs := make(map[string]string)

	// apply missing UUIDs
	for idx := range seeds {
		seed := &seeds[idx]
		switch strings.ToLower(seed.Kind) {
		case pkg.KindJobFunction:
			for i := range seed.Items {
				item := &seed.Items[i]
				jf := pkg.JobFunction{}
				err = item.Decode(&jf)
				if err != nil {
					panic(err)
				}
				if jf.UUID == "" {
					jf.UUID = uuid.NewString()
				}
				err = item.Encode(jf)
				if err != nil {
					panic(err)
				}
			}
		case pkg.KindTagCategory:
			for i := range seed.Items {
				item := &seed.Items[i]
				tc := pkg.TagCategory{}
				err = item.Decode(&tc)
				if err != nil {
					panic(err)
				}
				if tc.UUID == "" {
					tc.UUID = uuid.NewString()
				}
				for _, tag := range tc.Tags {
					if tag.UUID == "" {
						tag.UUID = uuid.NewString()
					}
				}
				err = item.Encode(tc)
				if err != nil {
					panic(err)
				}
			}
		case pkg.KindRuleSet:
			for i := range seed.Items {
				item := &seed.Items[i]
				rs := pkg.RuleSet{}
				err = item.Decode(&rs)
				if err != nil {
					panic(err)
				}
				rs.SeedDir = seed.Dir()
				if rs.UUID == "" {
					rs.UUID = uuid.NewString()
				}
				rulesetUUIDs[rs.Directory] = rs.UUID

				checksum, cErr := pkg.ChecksumDir(rs.Dir())
				if cErr != nil {
					panic(cErr)
				}
				rs.Checksum = fmt.Sprintf("%x", checksum)

				err = item.Encode(rs)
				if err != nil {
					panic(err)
				}
			}
		case pkg.KindTarget:
			for i := range seed.Items {
				item := &seed.Items[i]
				t := pkg.Target{}
				err = item.Decode(&t)
				if err != nil {
					panic(err)
				}
				if t.UUID == "" {
					t.UUID = uuid.NewString()
				}
				err = item.Encode(t)
				if err != nil {
					panic(err)
				}
			}
		default:
		}
		seedsByFile[seed.Filename()] = append(seedsByFile[seed.Filename()], seed)
	}

	// resolve ruleset dependencies
	for i := range seeds {
		seed := &seeds[i]
		switch strings.ToLower(seed.Kind) {
		case pkg.KindRuleSet:
			for j := range seed.Items {
				item := &seed.Items[j]
				rs := pkg.RuleSet{}
				err = item.Decode(&rs)
				if err != nil {
					panic(err)
				}

				for k, dep := range rs.Dependencies {
					if strings.HasPrefix(dep, "@") {
						u, found := rulesetUUIDs[dep[1:]]
						if !found {
							fmt.Printf("Could not resolve RuleSet depdendency `%s`\n", dep)
							continue
						}
						rs.Dependencies[k] = u
					}
				}
				err = item.Encode(rs)
				if err != nil {
					panic(err)
				}
			}
		default:
		}
	}

	for file, list := range seedsByFile {
		func() {
			f, fErr := os.Create(path.Join(outPath, file))
			if fErr != nil {
				panic(fErr)
			}
			defer f.Close()

			encoder := yaml.NewEncoder(f)
			for _, seed := range list {
				err = encoder.Encode(seed)
				if err != nil {
					panic(err)
				}
			}
			defer encoder.Close()
		}()
	}
}
