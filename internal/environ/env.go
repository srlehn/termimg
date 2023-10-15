package environ

import (
	"strings"

	"github.com/srlehn/termimg/internal/propkeys"
)

func EnvToProprietor(env []string) Properties {
	pr := NewProprietor()
	for _, v := range env {
		if len(v) == 0 {
			continue
		}
		var i int
		for i = 0; i < len(v)-1 && v[i] != '='; i++ {
		}
		pr.SetProperty(propkeys.EnvPrefix+v[:i], v[i+1:])
	}

	return pr
}

func CleanEnv(envParent, envInner []string) (env Properties) {
	commonVars := map[string]struct{}{
		`OLDPWD`:                      {},
		`PWD`:                         {},
		`PATH`:                        {},
		`PS2`:                         {},
		`PS3`:                         {},
		`PS4`:                         {},
		`SHLVL`:                       {},
		`_`:                           {},
		`LS_COLORS`:                   {},
		`LESS`:                        {},
		`LESSOPEN`:                    {},
		`MORE`:                        {},
		`SYSTEMD_LESS`:                {},
		`EDITOR`:                      {},
		`VISUAL`:                      {},
		`MANPATH`:                     {},
		`QT_ACCESSIBILITY`:            {},
		`QT_AUTO_SCREEN_SCALE_FACTOR`: {},
		`QT_SCALE_FACTOR`:             {},
		`GOPATH`:                      {},
		`GO111MODULE`:                 {},
		`DOCKER_HOST`:                 {},
		`ANDROID_HOME`:                {},
	}
	preserveVars := map[string]struct{}{
		`PS1`:              {},
		`TERM`:             {},
		`DISPLAY`:          {},
		`XDG_SESSION_TYPE`: {},
	}
	ep := EnvToProprietor(envParent)
	ei := EnvToProprietor(envInner)
	envCleaned := NewProprietor()
	for k, v := range ei.ExportProperties() {
		k, isEnvEntry := strings.CutPrefix(k, propkeys.EnvPrefix)
		if !isEnvEntry {
			continue
		}
		valOld, ok := ep.LookupEnv(k)
		if ok && valOld == v {
			if _, toPres := preserveVars[k]; !toPres {
				continue
			}
		}
		if _, isCommonVar := commonVars[k]; isCommonVar {
			continue
		}
		envCleaned.SetProperty(propkeys.EnvPrefix+k, v)
	}

	return envCleaned
}

func DetectChangedEnvVar(env, envCmp []string, names ...string) (name, value string) {
	// assume only 1 var will change
	valMap := make(map[string]string)
	valCmpMap := make(map[string]string)
	for _, e := range env {
		for _, name := range names {
			if val, found := strings.CutPrefix(e, name+`=`); found {
				valMap[name] = val
			}
		}
	}
	for _, e := range envCmp {
		for _, name := range names {
			if val, found := strings.CutPrefix(e, name+`=`); found {
				valCmpMap[name] = val
			}
		}
	}
	for _, n := range names {
		v, ok := valMap[n]
		vCmp, okCmp := valCmpMap[n]
		_, _ = ok, okCmp
		if v != vCmp {
			return n, v
		}
	}
	return ``, ``
}
