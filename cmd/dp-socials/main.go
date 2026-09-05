package main

import (
	"fmt"
	"sort"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func main() {
	names := make([]string, 0, len(game.Socials))
	for name := range game.Socials {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Println(name)
	}
}
