package routines

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/util"
	"golang.org/x/exp/slices"
)

func ListTermChecks() error {
	tm, err := termimg.Terminal()
	if err != nil {
		return err
	}
	defer tm.Close()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, tm.Name(), tm.Drawers()[0].Name())
	for k, v := range tm.Properties() {
		if !strings.HasPrefix(k, propkeys.CheckPrefix) {
			continue
		}
		fmt.Fprintf(w, "%s:\t%q\n", k, v)
	}

	return w.Flush()
}

func ListTermProps() error {
	tm, err := termimg.Terminal()
	if err != nil {
		return err
	}
	defer tm.Close()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "terminal:\t"+tm.Name())
	drawers := tm.Drawers()
	for i := range drawers {
		fmt.Fprintln(w, "drawer:\t"+drawers[i].Name())
	}
	fmt.Fprintln(w)
	prnt := func(key string) string {
		val, ok := tm.Property(key)
		if ok {
			return fmt.Sprintf("%q", val)
		}
		return ``
	}
	fmt.Fprintln(w, "query DA1 class:\t"+prnt(propkeys.DeviceClass))
	fmt.Fprintln(w, "query DA1 attrs:\t"+prnt(propkeys.DeviceAttributes))
	fmt.Fprintln(w, "query DA3 hex:\t"+prnt(`queryCache_G1s9MGM=`)) // DA3 hex
	fmt.Fprintln(w, "query DA3 ID:\t"+prnt(propkeys.DA3ID))

	fmt.Fprintln(w)
	var atLeastOneEnvVar bool
	for _, k := range util.MapsKeysSorted(tm.Properties()) {
		envName, found := strings.CutPrefix(k, propkeys.EnvPrefix)
		if !found {
			continue
		}
		v, _ := tm.LookupEnv(envName)
		fmt.Fprintf(w, "env %s:\t%q\n", envName, v)
		atLeastOneEnvVar = true
	}

	{
		passages, okPassages := tm.Property(propkeys.Passages)
		if okPassages && len(passages) > 0 {
			if atLeastOneEnvVar {
				fmt.Fprintln(w)
			}
			fmt.Fprintln(w, "muxers:\t"+passages)
		}
	}

	if tmw := tm.Window(); tmw != nil {
		fmt.Fprintln(w)
		if tmw.WindowFind() == nil {
			windowClass := tmw.WindowClass()
			fmt.Fprintln(w, "window name:\t"+tmw.WindowName())
			fmt.Fprintln(w, "window class:\t"+windowClass)
			fmt.Fprintln(w, "window instance:\t"+tmw.WindowInstance())

			var atLeastOneXRes bool
			if len(windowClass) > 0 {
				props := tm.Properties()
				keys := make([]string, len(props))
				for k := range props {
					if rest, found := strings.CutPrefix(k, propkeys.XResourcesPrefix+windowClass); found &&
						(strings.HasPrefix(rest, `.`) || strings.HasPrefix(rest, `*`)) {
						if !atLeastOneXRes {
							fmt.Fprintln(w)
							atLeastOneXRes = true
						}
						keys = append(keys, k)
					}
				}
				slices.Sort(keys)
				for _, k := range keys {
					if len(k) == 0 {
						continue
					}
					fmt.Fprintf(w, "x resource %s:\t%q\n", strings.TrimPrefix(k, propkeys.XResourcesPrefix), props[k])
				}
			}
		}
	}

	return w.Flush()
}
