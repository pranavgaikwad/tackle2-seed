package pkg

import (
	"crypto/sha256"
	"errors"
	"fmt"
	liberr "github.com/jortel/go-utils/error"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path"
	"strings"
)

const (
	AllVersions = 0
)

// Seedable kinds
const (
	KindJobFunction = "jobfunction"
	KindRuleSet     = "ruleset"
	KindTagCategory = "tagcategory"
	KindTarget      = "target"
)

// Seed document structure.
type Seed struct {
	path    string `yaml:"-"`
	Kind    string
	Version uint
	Items   []yaml.Node
}

// Filename returns the name of the file containing this Seed.
func (r *Seed) Filename() string {
	return path.Base(r.path)
}

// Dir returns the path to the directory that contains this Seed.
func (r *Seed) Dir() string {
	return path.Dir(r.path)
}

// DecodeItems decodes the yaml nodes of the Items slice on the Seed
// into their proper representations based on the Seed's Kind.
func (r *Seed) DecodeItems() (decoded []interface{}, err error) {
	for _, encoded := range r.Items {
		switch strings.ToLower(r.Kind) {
		case KindTagCategory:
			item := TagCategory{}
			err = encoded.Decode(&item)
			if err != nil {
				return
			}
			decoded = append(decoded, item)
		case KindJobFunction:
			item := JobFunction{}
			err = encoded.Decode(&item)
			if err != nil {
				return
			}
			decoded = append(decoded, item)
		case KindTarget:
			item := Target{SeedDir: r.Dir()}
			err = encoded.Decode(&item)
			decoded = append(decoded, item)
		case KindRuleSet:
			item := RuleSet{SeedDir: r.Dir()}
			err = encoded.Decode(&item)
			if err != nil {
				return
			}
			err = item.Load()
			if err != nil {
				return
			}
			decoded = append(decoded, item)
		default:
			err = liberr.New("unknown kind")
			return
		}
	}

	return
}

// ReadFromDir reads all seeds from the given directory.
func ReadFromDir(dir string, version uint) (seeds []Seed, checksum []byte, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		err = liberr.Wrap(err)
	}

	h := sha256.New()

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() {
			continue
		}
		s, sum, rErr := ReadFromFile(path.Join(dir, entry.Name()), version)
		if rErr != nil {
			err = rErr
			return
		}
		_, err = fmt.Fprint(h, sum)
		if err != nil {
			err = liberr.Wrap(err)
		}
		seeds = append(seeds, s...)
	}

	checksum = h.Sum(nil)
	return
}

// ReadFromFile reads all seeds from the given file.
func ReadFromFile(filePath string, version uint) (seeds []Seed, checksum []byte, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	defer f.Close()

	checksum, err = Checksum(f)
	if err != nil {
		return
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}

	decoder := yaml.NewDecoder(f)
	for {
		seed := Seed{path: filePath}
		err = decoder.Decode(&seed)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
				break
			}
			err = liberr.Wrap(err)
			return
		}
		if version == AllVersions || seed.Version == version {
			seeds = append(seeds, seed)
		}
	}
	return
}

// Checksum calculates a checksum for the contents of a reader.
func Checksum(r io.Reader) (sum []byte, err error) {
	h := sha256.New()
	_, err = io.Copy(h, r)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	sum = h.Sum(nil)
	return
}

// ChecksumDir calculates a checksum for the contents of a directory.
func ChecksumDir(dir string) (sum []byte, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		err = liberr.Wrap(err)
	}
	h := sha256.New()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		err = func() (fErr error) {
			f, fErr := os.Open(path.Join(dir, entry.Name()))
			if fErr != nil {
				fErr = liberr.Wrap(fErr)
				return
			}
			defer f.Close()
			chk, fErr := Checksum(f)
			if fErr != nil {
				fErr = liberr.Wrap(fErr)
				return
			}
			_, fErr = fmt.Fprint(h, chk)
			if fErr != nil {
				fErr = liberr.Wrap(fErr)
				return
			}
			return
		}()
	}
	sum = h.Sum(nil)
	return
}
