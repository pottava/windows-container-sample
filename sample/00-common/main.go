package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	key := os.Getenv("ENV_KEY")
	for _, env := range os.Environ() {
		if len(key) == 0 {
			fmt.Println(env)
			continue
		}
		keyval := strings.Split(env, "=")
		if len(keyval) <= 1 {
			continue
		}
		if !strings.EqualFold(key, keyval[0]) {
			continue
		}
		fmt.Println(keyval[1])
	}
}
