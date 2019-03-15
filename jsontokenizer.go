package gojsonsm

import (
	"errors"
	"fmt"
)

type tokenType int

const (
	tknUnknown tokenType = iota
	tknObjectStart
	tknObjectEnd
	tknObjectKeyDelim
	tknArrayStart
	tknArrayEnd
	tknListDelim
	tknString
	tknEscString
	tknInteger
	tknNumber
	tknNull
	tknTrue
	tknFalse
	tknEnd
)

func isLiteralToken(token tokenType) bool {
	return token >= tknString && token <= tknFalse
}

func tokenToText(token tokenType) string {
	switch token {
	case tknUnknown:
		return "unknown"
	case tknObjectStart:
		return "object_start"
	case tknObjectEnd:
		return "object_end"
	case tknObjectKeyDelim:
		return "object_key_delim"
	case tknArrayStart:
		return "array_start"
	case tknArrayEnd:
		return "array_end"
	case tknListDelim:
		return "list_delim"
	case tknString:
		return "string"
	case tknEscString:
		return "escaped_string"
	case tknInteger:
		return "integer"
	case tknNumber:
		return "number"
	case tknNull:
		return "null"
	case tknTrue:
		return "true"
	case tknFalse:
		return "false"
	case tknEnd:
		return "end"
	}
	return "??ERROR??"
}

type tokenizerState int

const (
	toksBeginValueOrEmpty tokenizerState = iota
	toksBeginValue
	toksBeginStringOrEmpty
	toksBeginString
	toksInString
	toksInStringEsc
	toksInStringEscU
	toksInStringEscU1
	toksInStringEscU12
	toksInStringEscU123
	toksNeg
	toks0
	toksT
	toksTr
	toksTru
	toksF
	toksFa
	toksFal
	toksFals
	toksN
	toksNu
	toksNul
	toks1
	toksDot
	toksDot0
	toksE
	toksESign
	toksE0
)

func tokIsSpaceChar(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

type jsonTokenizer struct {
	data []byte
	pos  int
}

func (tkn *jsonTokenizer) Reset(data []byte) {
	tkn.data = data
	tkn.pos = 0
}

func (tkn *jsonTokenizer) Position() int {
	return tkn.pos
}

func (tkn *jsonTokenizer) Seek(pos int) {
	tkn.pos = pos
}

func (tkn *jsonTokenizer) DataLoop(c byte, dataSlice []byte, startPos, dataPos, dataLen int, state tokenizerState, tokenType tokenType, strHasEscapes, numberIsNonInteger bool) (tokenType, error, byte, int, int, int, tokenizerState, bool, bool) {
DataLoop:
	for {
		if dataPos >= dataLen {
			// Due to the fact that numbers just kind of... end... during JSON parsing
			// we need some special logic to handle the cases where numbers are involved
			// and they are the only input to the parser...
			switch state {
			case toks1, toks0, toksDot0, toksE0:
				tokenType = tknNumber
				break DataLoop
			case toksBeginValue:
				tokenType = tknEnd
				break DataLoop
			default:
				// We couldn't have expected this... time to fail :/
				tokenType = tknUnknown
				err := errors.New("unexpected end of input")
				return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger
			}
		}

		c = dataSlice[dataPos]
		dataPos++

		switch state {
		case toksBeginValueOrEmpty:
			if c <= ' ' && tokIsSpaceChar(c) {
				startPos = dataPos
				continue DataLoop
			}
			if c == ']' {
				tokenType = tknArrayEnd
				break DataLoop
			}
			fallthrough

		case toksBeginValue:
			if c <= ' ' && tokIsSpaceChar(c) {
				startPos = dataPos
				continue DataLoop
			}

			switch c {
			case '{':
				tokenType = tknObjectStart
				break DataLoop
			case '}':
				tokenType = tknObjectEnd
				break DataLoop
			case ':':
				tokenType = tknObjectKeyDelim
				break DataLoop
			case '[':
				tokenType = tknArrayStart
				break DataLoop
			case ']':
				tokenType = tknArrayEnd
				break DataLoop
			case ',':
				tokenType = tknListDelim
				break DataLoop
			case '"':
				state = toksInString
				continue DataLoop
			case '-':
				state = toksNeg
				continue DataLoop
			case '0': // beginning of 0.123
				state = toks0
				continue DataLoop
			case 't': // beginning of true
				state = toksT
				continue DataLoop
			case 'f': // beginning of false
				state = toksF
				continue DataLoop
			case 'n': // beginning of null
				state = toksN
				continue DataLoop
			default:
				if '1' <= c && c <= '9' { // beginning of 1234.5
					state = toks1
					continue DataLoop
				}

				tokenType = tknUnknown
				err := fmt.Errorf("looking for beginning of value but found `%c`", c)
				return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger
			}

		case toksBeginStringOrEmpty:
			if c <= ' ' && tokIsSpaceChar(c) {
				startPos = dataPos
				continue DataLoop
			}
			if c == '}' {
				tokenType = tknObjectEnd
				break DataLoop
			}
			fallthrough

		case toksBeginString:
			if c <= ' ' && tokIsSpaceChar(c) {
				startPos = dataPos
				continue DataLoop
			}
			if c == '"' {
				state = toksInString
				continue DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("looking for beginning of object key string")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksInString:
			if c == '"' {
				tokenType = tknEscString
				break DataLoop
			}
			if c == '\\' {
				state = toksInStringEsc
				continue DataLoop
			}
			if c < 0x20 {

				tokenType = tknUnknown
				err := errors.New("in string literal")
				return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger
			}

			// continue with current state
			continue DataLoop

		case toksInStringEsc:
			strHasEscapes = true

			switch c {
			case 'b', 'f', 'n', 'r', 't', '\\', '/', '"':
				state = toksInString
				continue DataLoop
			case 'u':
				state = toksInStringEscU
				continue DataLoop
			default:
				tokenType = tknUnknown
				err := errors.New("in string escape code")
				return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger
			}

		case toksInStringEscU:
			if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
				state = toksInStringEscU1
				continue DataLoop
			}
			// numbers
			tokenType = tknUnknown
			err := errors.New("in \\u hexadecimal character escape")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksInStringEscU1:
			if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
				state = toksInStringEscU12
				continue DataLoop
			}
			// numbers
			tokenType = tknUnknown
			err := errors.New("in \\u hexadecimal character escape")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksInStringEscU12:
			if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
				state = toksInStringEscU123
				continue DataLoop
			}
			// numbers
			err := errors.New("in \\u hexadecimal character escape")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksInStringEscU123:
			if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
				state = toksInString
				continue DataLoop
			}
			// numbers
			err := errors.New("in \\u hexadecimal character escape")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksNeg:
			if c == '0' {
				state = toks0
				continue DataLoop
			}
			if '1' <= c && c <= '9' {
				state = toks1
				continue DataLoop
			}
			err := errors.New("in numeric literal")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toks1:
			if '0' <= c && c <= '9' {
				state = toks1
				continue DataLoop
			}
			fallthrough
		case toks0:
			if c == '.' {
				state = toksDot
				continue DataLoop
			}
			if c == 'e' || c == 'E' {
				state = toksE
				continue DataLoop
			}

			// need to rewind one character since this was non-numeric
			dataPos--
			tokenType = tknNumber
			break DataLoop

		case toksDot:
			numberIsNonInteger = true

			if '0' <= c && c <= '9' {
				state = toksDot0
				continue DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("after decimal point in numeric literal")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksDot0:
			if '0' <= c && c <= '9' {
				state = toksDot0
				continue DataLoop
			}
			if c == 'e' || c == 'E' {
				state = toksE
				continue DataLoop
			}

			// need to rewind one character since this was non-numeric
			dataPos--
			tokenType = tknNumber
			break DataLoop

		case toksE:
			numberIsNonInteger = true

			if c == '+' || c == '-' {
				state = toksESign
				continue DataLoop
			}
			fallthrough

		case toksESign:
			if '0' <= c && c <= '9' {
				state = toksE0
				continue DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("in exponent of numeric literal")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksE0:
			if '0' <= c && c <= '9' {
				// continue parsing numbers...
				continue DataLoop
			}

			// need to rewind one character since this was non-numeric
			dataPos--
			tokenType = tknNumber
			break DataLoop

		case toksT:
			if c == 'r' {
				state = toksTr
				continue DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("in literal true (expecting 'r')")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksTr:
			if c == 'u' {
				state = toksTru
				continue DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("in literal true (expecting 'u')")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksTru:
			if c == 'e' {
				tokenType = tknTrue
				break DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("in literal true (expecting 'e')")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksF:
			if c == 'a' {
				state = toksFa
				continue DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("in literal false (expecting 'a')")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksFa:
			if c == 'l' {
				state = toksFal
				continue DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("in literal false (expecting 'l')")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksFal:
			if c == 's' {
				state = toksFals
				continue DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("in literal false (expecting 's')")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksFals:
			if c == 'e' {
				tokenType = tknFalse
				break DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("in literal false (expecting 'e')")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksN:
			if c == 'u' {
				state = toksNu
				continue DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("in literal null (expecting 'u')")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksNu:
			if c == 'l' {
				state = toksNul
				continue DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("in literal null (expecting 'l')")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger

		case toksNul:
			if c == 'l' {
				tokenType = tknNull
				break DataLoop
			}
			tokenType = tknUnknown
			err := errors.New("in literal null (expecting 'l')")
			return tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger
		}
	}
	return tokenType, nil, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger
}

func sliceData(data []byte, beg, end int) []byte {
	return data[beg:end]
}

func (tkn *jsonTokenizer) Step() (tokenType, []byte, error) {
	// Bring everying local for optimization purposes
	dataSlice := tkn.data
	dataLen := len(dataSlice)
	dataPos := tkn.pos

	// Check that we aren't out of bounds...
	if dataPos >= dataLen {
		return tknEnd, nil, nil
	}

	// Keep track of where we started, so we can return the tokens data
	startPos := dataPos

	tokenType := tknUnknown
	state := toksBeginValue
	strHasEscapes := false
	numberIsNonInteger := false

	var c byte
	var err error

	tokenType, err, c, startPos, dataPos, dataLen, state, strHasEscapes, numberIsNonInteger = tkn.DataLoop(c, dataSlice, startPos, dataPos, dataLen, state, tokenType, strHasEscapes, numberIsNonInteger)
	if err != nil {
		return tokenType, nil, err
	}

	// Enhance some of the tokens with additional information
	if tokenType == tknNumber && !numberIsNonInteger {
		tokenType = tknInteger
	}
	if tokenType == tknEscString && !strHasEscapes {
		tokenType = tknString
	}

	endPos := dataPos
	tokenData := sliceData(tkn.data, startPos, endPos)

	// Update the scanners state
	tkn.pos = dataPos

	return tokenType, tokenData, nil

}
