package cpu

func ReadOp(data []byte, ip int, ops []Op) (Op, int) {
	if ip < 0 || ip >= len(data) {
		return nil, ip + 1
	}
	c = data[ip]
	if c < 0 || c > len(ops) {
		return nil, ip + 1
	}
	return ops[c], ip + 1
}
