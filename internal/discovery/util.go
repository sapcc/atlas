package discovery

func setMetricsLabelAndValue(t map[string]int, l string, i int) {
	if l != "" {
		if val, ok := t[l]; ok {
			t[l] = val + i
		} else {
			t[l] = i
		}
	}
}
