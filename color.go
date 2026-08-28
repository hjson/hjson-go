package hjson

// Style is the color style
type Style struct {
	Key, String, Number [2]string
	True, False, Null   [2]string
	Escape              [2]string
	Remark              [2]string
	Append              func(dst []byte, c byte) []byte
}

// TerminalStyle is for terminals
var TerminalStyle *Style

func init() {
	TerminalStyle = &Style{
		Key:    [2]string{"\x1B[94m", "\x1B[0m"},
		String: [2]string{"\x1B[92m", "\x1B[0m"},
		Number: [2]string{"\x1B[93m", "\x1B[0m"},
		True:   [2]string{"\x1B[96m", "\x1B[0m"},
		False:  [2]string{"\x1B[96m", "\x1B[0m"},
		Null:   [2]string{"\x1B[91m", "\x1B[0m"},
		Escape: [2]string{"\x1B[35m", "\x1B[0m"},
		Remark: [2]string{"\x1B[90m", "\x1B[0m"},
		Append: func(dst []byte, c byte) []byte {
			if c < ' ' && (c != '\r' && c != '\n' && c != '\t' && c != '\v') {
				dst = append(dst, "\\u00"...)
				dst = append(dst, hexp((c>>4)&0xF))
				return append(dst, hexp((c)&0xF))
			}
			return append(dst, c)
		},
	}
}
func hexp(p byte) byte {
	switch {
	case p < 10:
		return p + '0'
	default:
		return (p - 10) + 'a'
	}
}
