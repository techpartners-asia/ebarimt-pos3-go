package utils

import (
	"fmt"
	"time"
)

func Str2UnixTime(str string) int64 {
	// maybe change format
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		fmt.Println("Error parsing time:", err)
		return 0
	}

	// Convert to Unix time
	return t.Unix()
}
