package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ppawlinski/DiceyChess/assets"
)

func main() {
	assets.Init()
	ebiten.SetWindowSize(806, 606)
	ebiten.SetWindowTitle("Chess 2.0")
	c := NewChess()
	c.ui = c.CreateUI()
	if err := ebiten.RunGame(c); err != nil {
		log.Fatal(err)
	}
}
