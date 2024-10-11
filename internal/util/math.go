package util

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func GCD(m, n int) int {
	for n != 0 {
		m, n = n, m%n
	}
	return abs(m)
}

func LCM(m, n int) int {
	if n == 0 {
		return 0
	}
	return abs(n) * (abs(m) / GCD(m, n))
}
