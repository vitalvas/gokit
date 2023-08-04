package xstrings

func SliceContain(elems []string, key string) bool {
	for _, s := range elems {
		if s == key {
			return true
		}
	}
	return false
}
