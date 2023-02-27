package client

import (
	"flag"
	"fmt"
)

func EnumFlag(target *string, name string, safelist []string, usage string) {
	usageWithValues := fmt.Sprintf("%s, must be one of %v", usage, safelist)
	flag.Func(name, usageWithValues, func(flagValue string) error {
		for _, allowedValue := range safelist {
			if flagValue == allowedValue {
				*target = flagValue
				return nil
			}
		}
		return fmt.Errorf("must be one of %v", safelist)
	})
}
