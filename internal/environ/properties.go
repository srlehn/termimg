package environ

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/util"
)

type Properties interface {
	Enver
	PropertyExporter
	Property(key string) (string, bool)
	SetProperty(key, value string)
	MergeProperties(PropertyExporter)
	String() string
}

type Enver interface {
	Environ() []string
	LookupEnv(v string) (string, bool)
	Getenv(string) string
}

// https://pkg.go.dev/github.com/muesli/termenv#Environ
type müsliTermEnvEnviron interface {
	Environ() []string
	Getenv(string) string
}

var _ müsliTermEnvEnviron = (Enver)(nil)

type PropertyExporter interface {
	ExportProperties() map[string]string
}

var _ Properties = (*propertiesGeneric)(nil)

type propertiesGeneric struct {
	sync.Locker
	properties map[string]string
}

func NewProperties() Properties {
	return &propertiesGeneric{Locker: &sync.Mutex{}, properties: make(map[string]string)}
}

func CloneProperties(pr PropertyExporter) Properties {
	if pr == nil {
		return nil
	}
	p := &propertiesGeneric{Locker: &sync.Mutex{}, properties: make(map[string]string)}
	p.MergeProperties(pr)
	return p
}

// Property ...
func (p *propertiesGeneric) Property(key string) (string, bool) {
	if p == nil || p.properties == nil {
		*p = propertiesGeneric{Locker: &sync.Mutex{}, properties: make(map[string]string)}
		return ``, false
	}
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	v, ok := p.properties[key]
	return v, ok
}

// SetProperty ...
func (p *propertiesGeneric) SetProperty(key, value string) {
	if p == nil {
		panic(errors.NilReceiver())
	}
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	if p.properties == nil {
		p.properties = make(map[string]string)
	}
	p.properties[key] = value
}

func (p *propertiesGeneric) ExportProperties() map[string]string {
	if p == nil {
		panic(errors.NilReceiver())
	}
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	return p.properties
}

func (p *propertiesGeneric) LookupEnv(v string) (string, bool) {
	return p.Property(propkeys.EnvPrefix + v)
}

func (p *propertiesGeneric) Getenv(v string) string {
	s, ok := p.Property(propkeys.EnvPrefix + v)
	if ok {
		return s
	}
	return ``
}

func (p *propertiesGeneric) Environ() []string {
	if p == nil {
		panic(errors.NilReceiver())
	}
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	envSep := make([][2]string, 0, len(p.properties))
	for k, v := range p.properties {
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

// MergeProperties combines both Proprietors,
// possibly overwriting with values from pr.
func (p *propertiesGeneric) MergeProperties(pr PropertyExporter) {
	// TODO fix doc comment
	if p == nil || pr == nil {
		return
	}
	// p.SetProperties(pr.Properties())
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	if p.properties == nil {
		p.properties = make(map[string]string)
	}
	m := pr.ExportProperties()
	if m == nil {
		return
	}
	for k, v := range m {
		p.properties[k] = v
	}
}

func (p *propertiesGeneric) String() string {
	if p == nil || p.properties == nil {
		return `<nil>`
	}
	if p.Locker != nil {
		p.Lock()
		defer p.Unlock()
	}
	s := &strings.Builder{}
	_, _ = s.WriteString("properties: {\n")
	keysSorted := util.MapsKeysSorted(p.properties)
	for _, k := range keysSorted {
		_, _ = s.WriteString(fmt.Sprintf("\t\"%s\": %q\n", k, p.properties[k]))
	}
	_, _ = s.WriteString("}")
	return s.String()
}
