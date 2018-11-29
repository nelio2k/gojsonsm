// Copyright 2018 Couchbase, Inc. All rights reserved.

package gojsonsm

import (
	"bytes"
	"strings"
)

func reindentString(value string, indent string) string {
	return indent + strings.Replace(value, "\n", "\n"+indent, -1)
}

func DeepCopyStringArray(in []string) []string {
	if in == nil {
		return nil
	}

	out := make([]string, len(in))
	copy(out, in)
	return out
}

func StringSplitFirstInst(value, delim string) (ss []string) {
	if value == delim {
		ss = append(ss, delim)
		return
	}

	ss = strings.SplitAfterN(value, delim, 2)
	if len(ss) < 2 || ss[0] == delim {
		return
	}

	if ss[1] == "" && strings.HasSuffix(ss[0], delim) {
		ss[0] = strings.TrimSuffix(ss[0], delim)
		ss[1] = delim
	} else {
		ss = append(ss, "")
		ss[0] = strings.Trim(ss[0], delim)
		ss[2] = ss[1]
		ss[1] = delim
	}

	return
}

// flatten a string array into a string. each string element in string array is separated by a space
func FlattenStringArray(input []string) string {
	var output string
	if len(input) > 0 {
		var buffer bytes.Buffer
		for index, str := range input {
			if index != 0 {
				buffer.WriteString(" ")
			}
			buffer.WriteString(str)
		}
		output = buffer.String()
	}
	return output
}
