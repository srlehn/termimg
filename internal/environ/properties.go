package environ

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/util"
)

type Proprietor interface {
	Enver
	PropertyExporter
	Property(key string) (string, bool)
	SetProperty(key, value string)
	Merge(PropertyExporter)
	String() string
}

type Enver interface {
	Environ() []string
	LookupEnv(v string) (string, bool)
}

type PropertyExporter interface {
	Properties() map[string]string
}

var _ Proprietor = (*proprietorGeneric)(nil)

type proprietorGeneric struct {
	sync.Locker
	propsExtra map[string]string
}

func NewProprietor() Proprietor {
	return &proprietorGeneric{Locker: &sync.Mutex{}, propsExtra: make(map[string]string)}
}

func CloneProprietor(pr PropertyExporter) Proprietor {
	if pr == nil {
		return nil
	}
	p := &proprietorGeneric{Locker: &sync.Mutex{}, propsExtra: make(map[string]string)}
	if pr == nil {
		return nil
	}
	p.Merge(pr)
	return p
}

// Property ...
func (p *proprietorGeneric) Property(key string) (string, bool) {
	if p == nil || p.propsExtra == nil {
		p = &proprietorGeneric{Locker: &sync.Mutex{}, propsExtra: make(map[string]string)}
		return ``, false
	}
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	v, ok := p.propsExtra[key]
	return v, ok
}

// SetProperty ...
func (p *proprietorGeneric) SetProperty(key, value string) {
	if p == nil {
		p = &proprietorGeneric{Locker: &sync.Mutex{}, propsExtra: make(map[string]string)}
	}
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	if p.propsExtra == nil {
		p.propsExtra = make(map[string]string)
	}
	p.propsExtra[key] = value
}

func (p *proprietorGeneric) Properties() map[string]string {
	if p == nil {
		p = &proprietorGeneric{Locker: &sync.Mutex{}, propsExtra: make(map[string]string)}
	}
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	return p.propsExtra
}

func (p *proprietorGeneric) LookupEnv(v string) (string, bool) {
	return p.Property(propkeys.EnvPrefix + v)
}

func (p *proprietorGeneric) Environ() []string {
	if p == nil {
		p = &proprietorGeneric{Locker: &sync.Mutex{}, propsExtra: make(map[string]string)}
	}
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	envSep := make([][2]string, 0, len(p.propsExtra))
	for k, v := range p.propsExtra {
		after, found := strings.CutPrefix(k, propkeys.EnvPrefix)
		if !found {
			continue
		}
		envSep = append(envSep, [2]string{after, v})
	}
	sort.Slice(envSep, func(i, j int) bool { return envSep[i][0] < envSep[j][0] })
	env := make([]string, 0, len(envSep))
	for _, entry := range envSep {
		env = append(env, entry[0]+`=`+entry[1])
	}
	return env
}

// Merge combines both Proprietors,
// possibly overwriting with values from pr.
func (p *proprietorGeneric) Merge(pr PropertyExporter) {
	// TODO fix doc comment
	if p == nil || pr == nil {
		return
	}
	// p.SetProperties(pr.Properties())
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	if p.propsExtra == nil {
		p.propsExtra = make(map[string]string)
	}
	m := pr.Properties()
	if m == nil {
		return
	}
	for k, v := range m {
		p.propsExtra[k] = v
	}
}

func (p *proprietorGeneric) String() string {
	if p == nil || p.propsExtra == nil {
		return `<nil>`
	}
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	s := &strings.Builder{}
	_, _ = s.WriteString("properties: {\n")
	keysSorted := util.MapsKeysSorted(p.propsExtra)
	for _, k := range keysSorted {
		_, _ = s.WriteString(fmt.Sprintf("\t\"%s\": %q\n", k, p.propsExtra[k]))
	}
	_, _ = s.WriteString("}")
	return s.String()
}
