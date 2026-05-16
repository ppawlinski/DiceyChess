package game

import "math/rand"

type Dice struct {
	rng  *rand.Rand
	seed int64
}

func NewDice(seed int64) *Dice {
	return &Dice{
		rng:  rand.New(rand.NewSource(seed)),
		seed: seed,
	}
}

func (d *Dice) Roll() int {
	return d.rng.Intn(6) + 1
}

func (d *Dice) RollTwo() (int, int) {
	return d.Roll(), d.Roll()
}
