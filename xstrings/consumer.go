package xstrings

import (
	"fmt"
	"os"
	"time"
)

func GetConsumerTag() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown", err
	}

	return fmt.Sprintf("%s-%x", hostname, time.Now().UnixNano()), nil
}
