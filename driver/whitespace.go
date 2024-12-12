package driver

func notWhitespace(sql string) bool {
	const (
		code = iota
		minus
		slash
		ccomment
		endcomment
		sqlcomment
	)

	state := code
	for _, b := range ([]byte)(sql) {
		if b == 0 {
			break
		}

		switch state {
		case code:
			switch b {
			case '-':
				state = minus
			case '/':
				state = slash
			case ' ', ';', '\t', '\n', '\v', '\f', '\r':
				continue
			default:
				return true
			}
		case minus:
			if b != '-' {
				return true
			}
			state = sqlcomment
		case slash:
			if b != '*' {
				return true
			}
			state = ccomment
		case ccomment:
			if b == '*' {
				state = endcomment
			}
		case endcomment:
			switch b {
			case '/':
				state = code
			case '*':
				state = endcomment
			default:
				state = ccomment
			}
		case sqlcomment:
			if b == '\n' {
				state = code
			}
		}
	}

	switch state {
	case code, ccomment, endcomment, sqlcomment:
		return false
	default:
		return true
	}
}
