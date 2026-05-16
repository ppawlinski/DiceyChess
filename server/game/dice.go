package game

import "math/rand"

type Dice struct {
	rng *rand.Rand
}

func NewDice(seed int64) *Dice {
	return &Dice{
		rng: rand.New(rand.NewSource(seed)),
	}
}

func (d *Dice) Roll() int {
	return d.rng.Intn(6) + 1
}

func (d *Dice) RollTwo() (int, int) {
	return d.Roll(), d.Roll()
}
