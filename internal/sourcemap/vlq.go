package sourcemap

const vlqCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

var vlqDecode [256]int

func init() {
	for i := range vlqDecode {
		vlqDecode[i] = -1
	}
	for i := 0; i < len(vlqCharset); i++ {
		vlqDecode[vlqCharset[i]] = i
	}
}

func decodeVLQ(s string, i int) (n int, next int, ok bool) {
	shift := 0
	value := 0
	for i < len(s) {
		d := vlqDecode[s[i]]
		i++
		if d < 0 {
			return 0, i, false
		}
		value += (d & 31) << shift
		if d&32 == 0 {
			if value&1 != 0 {
				n = ^(value >> 1)
			} else {
				n = value >> 1
			}
			return n, i, true
		}
		shift += 5
	}
	return 0, i, false
}

func encodeVLQ(n int) string {
	u := uint(n) << 1
	if n < 0 {
		u = uint(^n)<<1 | 1
	}
	var b []byte
	for {
		digit := int(u & 31)
		u >>= 5
		if u > 0 {
			digit |= 32
		}
		b = append(b, vlqCharset[digit])
		if u == 0 {
			break
		}
	}
	return string(b)
}
