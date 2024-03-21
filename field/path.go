package field

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/openimsdk/tools/errs"
)

// Path represents the path from some root to a particular field.
type Path struct {
	name   string // the name of this field or "" if this is an index
	index  string // if name == "", this is a subscript (index or map key) of the previous element
	parent *Path  // nil if this is the root element
}

// NewPath creates a root Path object.
func NewPath(name string, moreNames ...string) *Path {
	r := &Path{name: name, parent: nil}
	for _, anotherName := range moreNames {
		r = &Path{name: anotherName, parent: r}
	}
	return r
}

// return absolude path join ../config/, this is k8s container config path.
func GetDefaultConfigPath() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", errs.WrapMsg(err, "failed to get executable path")
	}

	configPath, err := OutDir(filepath.Join(filepath.Dir(executablePath), "../config/"))
	if err != nil {
		return "", errs.WrapMsg(err, "failed to get output directory", "outDir", filepath.Join(filepath.Dir(executablePath), "../config/"))
	}
	return configPath, nil
}

// getProjectRoot returns the absolute path of the project root directory.
func GetProjectRoot() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", errs.Wrap(err)
	}
	projectRoot, err := OutDir(filepath.Join(filepath.Dir(executablePath), "../../../../.."))
	if err != nil {
		return "", errs.Wrap(err)
	}
	return projectRoot, nil
}

// Root returns the root element of this Path.
func (p *Path) Root() *Path {
	for ; p.parent != nil; p = p.parent {
		// Do nothing.
	}
	return p
}

// Child creates a new Path that is a child of the method receiver.
func (p *Path) Child(name string, moreNames ...string) *Path {
	r := NewPath(name, moreNames...)
	r.Root().parent = p
	return r
}

// Index indicates that the previous Path is to be subscripted by an int.
// This sets the same underlying value as Key.
func (p *Path) Index(index int) *Path {
	return &Path{index: strconv.Itoa(index), parent: p}
}

// Key indicates that the previous Path is to be subscripted by a string.
// This sets the same underlying value as Index.
func (p *Path) Key(key string) *Path {
	return &Path{index: key, parent: p}
}

// String produces a string representation of the Path.
func (p *Path) String() string {
	// make a slice to iterate
	elems := []*Path{}
	for ; p != nil; p = p.parent {
		elems = append(elems, p)
	}

	// iterate, but it has to be backwards
	buf := bytes.NewBuffer(nil)
	for i := range elems {
		p := elems[len(elems)-1-i]
		if p.parent != nil && len(p.name) > 0 {
			// This is either the root or it is a subscript.
			buf.WriteString(".")
		}
		if len(p.name) > 0 {
			buf.WriteString(p.name)
		} else {
			fmt.Fprintf(buf, "[%s]", p.index)
		}
	}
	return buf.String()
}
