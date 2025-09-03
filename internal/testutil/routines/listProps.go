package routines

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/util"
	"github.com/srlehn/termimg/term"
)

func ListTermChecks() error {
	tm, err := termimg.Terminal()
	if err != nil {
		return err
	}
	defer tm.Close()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, tm.Name(), tm.Drawers()[0].Name())
	for k, v := range tm.ExportProperties() {
		if !strings.HasPrefix(k, propkeys.CheckPrefix) {
			continue
		}
		fmt.Fprintf(w, "%s:\t%q\n", k, v)
	}

	return w.Flush()
}

func ListTermProps(tm *term.Terminal, listTerm, listDrawers, listQueries, listEnv, listPassages, listWindow, listXRes bool) error {
	var singleType bool
	{
		var typeCount int
		for _, tp := range []bool{listTerm, listDrawers, listQueries, listEnv, listPassages, listWindow, listXRes} {
			if tp {
				typeCount++
			}
		}
		if typeCount == 1 {
			singleType = true
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	var needsSep bool
	prSepNL := func() {
		if needsSep {
			fmt.Fprintln(w)
		}
		needsSep = false
	}
	if listTerm {
		if singleType {
			fmt.Fprintln(w, tm.Name())
		} else {
			fmt.Fprintln(w, "terminal:\t"+tm.Name())
		}
		needsSep = true
	}

	if listDrawers {
		var atLeastOneDrawer bool
		var prefix string
		if !singleType {
			prefix = "drawer:\t"
		}
		drawers := tm.Drawers()
		for _, dr := range drawers {
			if dr == nil {
				continue
			}
			prSepNL()
			atLeastOneDrawer = true
			fmt.Fprintln(w, prefix+dr.Name())
		}
		needsSep = atLeastOneDrawer
	}

	if listQueries {
		var prefix string
		if !singleType {
			prefix = "query "
		}
		prSepNL()
		pr := func(name, key string) {
			value, ok := tm.Property(key)
			if !ok || len(value) == 0 {
				return
			}
			fmt.Fprintf(w, "%s%s:\t%q\n", prefix, name, value)
		}
		pr(`DA1 class`, propkeys.DeviceClass)
		pr(`DA1 attrs`, propkeys.DeviceAttributes)
		pr(`DA2 model letter`, propkeys.DA2ModelLetter)
		pr(`DA2 model`, propkeys.DA2Model)
		pr(`DA2 version`, propkeys.DA2Version)
		pr(`DA3 ID`, propkeys.DA3ID)
		pr(`DA3 ID hex`, propkeys.DA3IDHex)
		pr(`XTVERSION`, propkeys.XTVERSION)
		pr(`XTGETTCAP(TN)`, propkeys.XTGETTCAPSpecialTN)
		needsSep = true
	}

	if listEnv {
		var atLeastOneEnvVar bool
		var prefix string
		if !singleType {
			prefix = "env "
		}
		for _, k := range util.MapsKeysSorted(tm.ExportProperties()) {
			envName, found := strings.CutPrefix(k, propkeys.EnvPrefix)
			if !found {
				continue
			}
			prSepNL()
			v, _ := tm.LookupEnv(envName)
			fmt.Fprintf(w, prefix+"%s:\t%q\n", envName, v)
			atLeastOneEnvVar = true
		}
		needsSep = atLeastOneEnvVar
	}

	if listPassages {
		var prefix string
		if !singleType {
			prefix = "muxers:\t"
		}
		passages, okPassages := tm.Property(propkeys.Passages)
		if okPassages && len(passages) > 0 {
			prSepNL()
			fmt.Fprintln(w, prefix+passages)
			needsSep = true
		}
	}

	if listWindow || listXRes {
		if tmw := tm.Window(); tmw != nil && tmw.WindowFind() == nil {
			windowClass := tmw.WindowClass()
			if listWindow {
				prSepNL()
				fmt.Fprintln(w, "window name:\t"+tmw.WindowName())
				fmt.Fprintln(w, "window class:\t"+windowClass)
				fmt.Fprintln(w, "window instance:\t"+tmw.WindowInstance())
				needsSep = true
			}

			if listXRes {
				var prefix string
				if !singleType {
					prefix = "x resource "
				}
				if len(windowClass) > 0 {
					props := tm.ExportProperties()
					keys := make([]string, len(props))
					for k := range props {
						if rest, found := strings.CutPrefix(k, propkeys.XResourcesPrefix+windowClass); found &&
							(strings.HasPrefix(rest, `.`) || strings.HasPrefix(rest, `*`)) {
							prSepNL()
							keys = append(keys, k)
						}
					}
					slices.Sort(keys)
					for _, k := range keys {
						if len(k) == 0 {
							continue
						}
						fmt.Fprintf(w, prefix+"%s:\t%q\n", strings.TrimPrefix(k, propkeys.XResourcesPrefix), props[k])
					}
				}
			}
		}
	}

	return w.Flush()
}
