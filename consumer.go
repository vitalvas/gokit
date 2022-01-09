package gokit

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

	return fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano()), nil
}
