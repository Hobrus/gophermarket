package luhn

// IsValid checks if the provided number is valid according to the Luhn algorithm.
func IsValid(number string) bool {
	if len(number) < 2 {
		return false
	}

	var sum int
	double := false
	for i := len(number) - 1; i >= 0; i-- {
		c := number[i]
		if c < '0' || c > '9' {
			return false
		}
		d := int(c - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}
