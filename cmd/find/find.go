package main

import (
	"fmt"
	"time"

	"github.com/mpsalisbury/cards/pkg/discovery"
)

func main() {
	locs, err := discovery.FindService(time.Second)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	for _, loc := range locs {
		fmt.Printf("Found HeartsServer at %s\n", loc)
	}
}
