package entities

type Food struct {
	Energetic
}

func NewFood(amt int) *Food {
	return &Food{Energy(amt)}
}

func (f *Food) Rune() rune {
	return '.'
}
