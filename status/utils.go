package status

func BytesToMiB(b uint64) float64 {
	return float64(b) / (1024 * 1024)
}
