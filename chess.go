package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math/rand/v2"
	"strconv"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	chessInput "github.com/ppawlinski/DiceyChess/input"
	"golang.org/x/image/font/gofont/goregular"
)

type Chess struct {
	board         *Board
	state         *GameState
	input         *chessInput.Input
	ui            *ebitenui.UI
	budgetCounter *widget.Text
}

func NewChess() *Chess {
	chess := Chess{
		board: NewBoard(),
		state: NewGameState(),
		input: chessInput.NewInput(),
	}
	chess.board.Reset()
	return &chess
}

func (c *Chess) CreateUI() *ebitenui.UI {
	face, _ := loadFont(15)
	endTurnButtonImage, _ := loadButtonImage()
	var rollButton *widget.Button
	var endTurnButton *widget.Button
	var budgetCounter *widget.Text
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewAnchorLayout(
				widget.AnchorLayoutOpts.Padding(
					widget.NewInsetsSimple(10),
				),
			),
		),
	)
	endTurnButton = widget.NewButton(
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(
				widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionEnd,
					VerticalPosition:   widget.AnchorLayoutPositionEnd,
				},
			),
			func(w *widget.Widget) { w.Visibility = widget.Visibility_Hide },
		),
		widget.ButtonOpts.Image(endTurnButtonImage),
		widget.ButtonOpts.Text(
			"End turn",
			face,
			&widget.ButtonTextColor{
				Idle: color.RGBA{0, 0, 0, 255},
			},
		),
		widget.ButtonOpts.TextPadding(widget.Insets{
			Left:   30,
			Right:  30,
			Top:    5,
			Bottom: 5,
		}),
		widget.ButtonOpts.ClickedHandler(
			func(args *widget.ButtonClickedEventArgs) {
				rollButton.GetWidget().Visibility = widget.Visibility_Show
				endTurnButton.GetWidget().Visibility = widget.Visibility_Hide
				c.state.leftoverBudget[c.state.colorToMove] = c.state.currentBudget > 0
				c.state.colorToMove = (c.state.colorToMove + 1) % 2
				c.state.currentBudget = 0
				c.state.turnStarted = false
				budgetCounter.Label = strconv.Itoa(c.state.currentBudget)
			},
		),
	)

	rollButton = widget.NewButton(
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(
				widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionEnd,
					VerticalPosition:   widget.AnchorLayoutPositionCenter,
				},
			),
		),
		widget.ButtonOpts.Image(endTurnButtonImage),
		widget.ButtonOpts.Text(
			"Roll",
			face,
			&widget.ButtonTextColor{
				Idle: color.RGBA{0, 0, 0, 255},
			},
		),
		widget.ButtonOpts.TextPadding(widget.Insets{
			Left:   30,
			Right:  30,
			Top:    30,
			Bottom: 30,
		}),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			rollButton.GetWidget().Visibility = widget.Visibility_Hide
			endTurnButton.GetWidget().Visibility = widget.Visibility_Show
			c.state.turnStarted = true
			c.state.currentBudget = rand.IntN(6) + 1
			if c.state.leftoverBudget[c.state.colorToMove] {
				c.state.currentBudget++
			}
			budgetCounter.Label = strconv.Itoa(c.state.currentBudget)
		}),
	)

	budgetContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 200, 255, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(
				widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionEnd,
					VerticalPosition:   widget.AnchorLayoutPositionStart,
				},
			),
			widget.WidgetOpts.MinSize(100, 100),
		),
	)
	budgetLabel := widget.NewText(
		widget.TextOpts.Text("Current budget:", face, color.White),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(
				widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionCenter,
					VerticalPosition:   widget.AnchorLayoutPositionStart,
				},
			),
		),
	)
	budgetCounter = widget.NewText(
		widget.TextOpts.Text("0", face, color.White),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(
				widget.AnchorLayoutData{
					HorizontalPosition: widget.AnchorLayoutPositionCenter,
					VerticalPosition:   widget.AnchorLayoutPositionEnd,
				},
			),
		),
	)
	c.budgetCounter = budgetCounter
	budgetContainer.AddChild(budgetLabel)
	budgetContainer.AddChild(budgetCounter)

	rootContainer.AddChild(budgetContainer)
	rootContainer.AddChild(endTurnButton)
	rootContainer.AddChild(rollButton)

	return &ebitenui.UI{
		Container: rootContainer,
	}
}

func (c *Chess) Update() error {
	c.ui.Update()
	if !c.state.turnStarted {
		return nil
	}

	lmbEvent := c.input.GetButtonEvent(chessInput.LMB)
	if lmbEvent == chessInput.Click {
		if c.state.selectedPiece.Undefined() {
			c.board.SelectPiece(c.state)
			if !c.state.selectedPiece.Undefined() {
				c.state.dragging = true
				c.state.possibleMoves = c.board.GetPossibleMoves(c.state.selectedPiece)
			}
		} else {
			selectedMove := c.board.HitCheck(c.state)
			if selectedMove.Valid() {
				if c.board.GetPiece(selectedMove) != nil && c.board.GetColor(selectedMove) == c.state.colorToMove {
					c.board.SelectPiece(c.state)
					c.state.dragging = true
					c.state.possibleMoves = c.board.GetPossibleMoves(c.state.selectedPiece)
				} else {
					c.board.MoveSelected(c.state, selectedMove)
					c.state.selectedPiece.Reset()
					c.state.possibleMoves = nil
				}
			} else {
				c.state.selectedPiece.Reset()
				c.state.possibleMoves = nil
			}
		}
	} else if lmbEvent == chessInput.Hold && c.state.dragging {
		c.board.GetPiece(c.state.selectedPiece).Piece().SetDragOffset(ebiten.CursorPosition())
	} else if lmbEvent == chessInput.Release {
		if c.state.dragging {
			selectedMove := c.board.HitCheck(c.state)
			c.board.MoveSelected(c.state, selectedMove)
		}
		c.state.dragging = false
	}
	c.budgetCounter.Label = strconv.Itoa(c.state.currentBudget)
	return nil
}

func (c *Chess) Draw(screen *ebiten.Image) {
	c.board.Draw(screen, c.state)
	c.ui.Draw(screen)
}

func (g *Chess) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 806, 606 //!!!3PPA TODO embed it in some UI element
}
func loadFont(size float64) (text.Face, error) {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Fatal(err)
		return nil, fmt.Errorf("Error loading font: %w", err)
	}

	return &text.GoTextFace{
		Source: s,
		Size:   size,
	}, nil
}

func loadButtonImage() (*widget.ButtonImage, error) {

	idle := image.NewBorderedNineSliceColor(color.NRGBA{90, 230, 76, 255}, color.NRGBA{90, 90, 90, 255}, 3)

	hover := image.NewBorderedNineSliceColor(color.NRGBA{70, 170, 76, 255}, color.NRGBA{70, 70, 70, 255}, 3)

	pressed := image.NewAdvancedNineSliceColor(color.NRGBA{70, 120, 96, 255}, image.NewBorder(3, 2, 2, 2, color.NRGBA{70, 70, 70, 255}))

	return &widget.ButtonImage{
		Idle:    idle,
		Hover:   hover,
		Pressed: pressed,
	}, nil
}
