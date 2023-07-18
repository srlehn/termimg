package routines

import (
	"fmt"
	"strings"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/util"
)

func ListTermChecks() error {
	tm, err := termimg.Terminal()
	if err != nil {
		return err
	}
	defer tm.Close()
	fmt.Println(tm.Name(), tm.Drawers()[0].Name())
	for k, v := range tm.Properties() {
		if !strings.HasPrefix(k, propkeys.CheckPrefix) {
			continue
		}
		fmt.Printf("%s:\t%q\n", k, v)
	}

	return nil
}

func ListTermProps() error {
	tm, err := termimg.Terminal()
	if err != nil {
		return err
	}
	defer tm.Close()
	fmt.Println("terminal", tm.Name())
	for i := range tm.Drawers() {
		fmt.Println("drawer", tm.Drawers()[i].Name())
	}
	fmt.Println()
	prnt := func(key string) string {
		val, ok := tm.Property(key)
		if ok {
			return fmt.Sprintf("%q", val)
		}
		return ``
	}
	fmt.Println("query DA1 class:", prnt(propkeys.DeviceClass))
	fmt.Println("query DA1 attrs:", prnt(propkeys.DeviceAttributes))
	fmt.Println("query DA3 hex:", prnt(`queryCache_G1s9MGM=`)) // DA3 hex
	fmt.Println("query DA3 ID:", prnt(propkeys.DA3ID))

	fmt.Println()
	for _, k := range util.MapsKeysSorted(tm.Properties()) {
		envName, found := strings.CutPrefix(k, propkeys.EnvPrefix)
		if !found {
			continue
		}
		v, _ := tm.LookupEnv(envName)
		fmt.Printf("env %s:\t%q\n", envName, v)
	}

	{
		passages, okPassages := tm.Property(propkeys.Passages)
		if okPassages && len(passages) > 0 {
			fmt.Println()
			fmt.Println("muxers: ", passages)
		}
	}

	if tmw := tm.Window(); tmw != nil {
		fmt.Println()
		if tmw.WindowFind() == nil {
			fmt.Println("window name:", tmw.WindowName())
			fmt.Println("window class:", tmw.WindowClass())
			fmt.Println("window instance:", tmw.WindowInstance())
		}
	}

	return nil
}
