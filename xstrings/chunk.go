package xstrings

func Chunks(lst []string, chunkSize int) [][]string {
	var result [][]string

	for i := 0; i < len(lst); i += chunkSize {
		end := i + chunkSize
		if end > len(lst) {
			end = len(lst)
		}
		result = append(result, lst[i:end])
	}

	return result
}
