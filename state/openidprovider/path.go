package openidprovider

import (
	"fmt"
	"strconv"
	"unicode"
)

func parseRolePath(str []rune) ([]string, []rune, error) {
	ids := make([]string, 0)

	for {
		var id string
		var err error
		if str[0] == '"' {
			id, str, err = quoted(str)
		} else {
			id, str, err = unquoted(str)
		}
		if err != nil {
			return nil, nil, err
		}

		if id == "" {
			return nil, nil, fmt.Errorf("empty identifier")
		}

		ids = append(ids, id)

		if len(str) == 0 {
			break
		} else if str[0] != '.' {
			return nil, nil, fmt.Errorf("unexpected '%c', expected '.'", str[0])
		} else {
			str = str[1:]
		}
	}

	return ids, str, nil
}

func isIdStart(r rune) bool {
	return unicode.In(
		r,
		unicode.L,
		unicode.Nl,
		unicode.Other_ID_Start,
	) && !unicode.In(
		r,
		unicode.Pattern_Syntax,
		unicode.Pattern_White_Space,
	)
}

func isIdContinue(r rune) bool {
	return unicode.In(
		r,
		unicode.L,
		unicode.Nl,
		unicode.Other_ID_Start,
		unicode.Mn,
		unicode.Mc,
		unicode.Nd,
		unicode.Pc,
		unicode.Other_ID_Continue,
	) && !unicode.In(
		r,
		unicode.Pattern_Syntax,
		unicode.Pattern_White_Space,
	)
}

func unquoted(str []rune) (string, []rune, error) {
	r := str[0]
	if r != '$' && r != '_' && !isIdStart(r) {
		return "", str, nil
	}

	idx := 1
	for _, r := range str[1:] {
		if r != '$' && r != '_' && r != '\u200c' && r != '\u200d' && !isIdContinue(r) {
			break
		} else {
			idx++
		}
	}

	id := string(str[:idx])
	str = str[idx:]
	return id, str, nil
}

func quoted(str []rune) (string, []rune, error) {
	str = str[1:]

	var id string
	for len(str) > 0 {
		r := str[0]
		str = str[1:]

		switch r {
		case '"':
			return id, str, nil
		case '\\':
			var esc string
			var err error
			esc, str, err = escaped(str)
			if err != nil {
				return "", nil, err
			}
			id += esc
		default:
			id += string(r)
		}
	}

	return "", nil, fmt.Errorf("unexpected end of input in quoted string '%s'", id)
}

func escaped(str []rune) (string, []rune, error) {
	if len(str) < 2 {
		return "", nil, fmt.Errorf("unexpected end of input, expected escape sequence")
	}

	r := str[1]
	str = str[2:]

	switch r {
	case '"':
		return `"`, str, nil
	case '\\':
		return `\`, str, nil
	case '/':
		return "/", str, nil
	case 'b':
		return "\b", str, nil
	case 'f':
		return "\f", str, nil
	case 'n':
		return "\n", str, nil
	case 'r':
		return "\r", str, nil
	case 't':
		return "\t", str, nil
	case 'u':
		return codepoint(str)
	default:
		return "", nil, fmt.Errorf(`unknown escape sequence '\%c'`, r)
	}
}

func codepoint(str []rune) (string, []rune, error) {
	if len(str) < 4 {
		return "", nil, fmt.Errorf("unexpected end of input, expected unicode codepoint")
	}

	hex := string(str[:4])
	str = str[4:]

	n, err := strconv.ParseUint(hex, 16, 16)
	if err != nil {
		return "", nil, fmt.Errorf("invalid unicode codepoint '%s'", hex)
	}

	return string(rune(n)), str, nil
}
