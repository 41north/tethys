package util

import "fmt"

func ElideString(str string) string {
	length := len(str)
	if length <= 16 {
		return str
	}
	return fmt.Sprintf("%s...%s", str[:6], str[length-4:])
}
