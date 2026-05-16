package assets

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/ppawlinski/DiceyChess/config"
)

type Assets struct {
	DarkSquare        *ebiten.Image
	LightSquare       *ebiten.Image
	HighlightedSquare *ebiten.Image
	WhiteKing         *ebiten.Image
	BlackKing         *ebiten.Image
	WhiteQueen        *ebiten.Image
	BlackQueen        *ebiten.Image
	WhiteBishop       *ebiten.Image
	BlackBishop       *ebiten.Image
	WhiteKnight       *ebiten.Image
	BlackKnight       *ebiten.Image
	WhiteRook         *ebiten.Image
	BlackRook         *ebiten.Image
	WhitePawn         *ebiten.Image
	BlackPawn         *ebiten.Image
}

var Images Assets

func Init() {
	image := ebiten.NewImage(config.TileSize, config.TileSize)
	image.Fill(color.RGBA{117, 59, 12, 255})
	Images.DarkSquare = image
	image = ebiten.NewImage(config.TileSize, config.TileSize)
	image.Fill(color.RGBA{196, 151, 114, 255})
	Images.LightSquare = image
	image = ebiten.NewImage(config.TileSize, config.TileSize)
	image.Fill(color.RGBA{255, 127, 80, 255})
	Images.HighlightedSquare = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\pawn.png")
	Images.BlackPawn = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\pawnWhite.png")
	Images.WhitePawn = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\king.png")
	Images.BlackKing = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\kingWhite.png")
	Images.WhiteKing = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\queen.png")
	Images.BlackQueen = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\queenWhite.png")
	Images.WhiteQueen = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\bishop.png")
	Images.BlackBishop = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\bishopWhite.png")
	Images.WhiteBishop = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\knight.png")
	Images.BlackKnight = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\knightWhite.png")
	Images.WhiteKnight = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\rook.png")
	Images.BlackRook = image
	image, _, _ = ebitenutil.NewImageFromFile("images\\rookWhite.png")
	Images.WhiteRook = image
}
