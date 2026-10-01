package xflags

import (
	"os"
	"strings"
)

// envValue returns the option's value from its environment variable, if the
// variable is set and non-empty.
func (opt *Option) envValue() (string, bool) {
	if opt.Env == "" {
		return "", false
	}
	val, ok := os.LookupEnv(opt.Env)
	if !ok || val == "" {
		return "", false
	}
	return val, true
}

// applyEnvMulti splits a raw environment value for slice and map options so a
// single variable can seed multiple entries. For scalar options the raw value
// is returned unchanged.
func (opt *Option) applyEnvMulti(raw string) error {
	if opt.choice == choiceAppend || opt.choice == choiceMerge {
		for part := range strings.SplitSeq(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if err := opt.applyString(part); err != nil {
				return err
			}
		}
		return nil
	}
	return opt.applyString(raw)
}
