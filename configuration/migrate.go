package configuration

type Migrate struct {
	DB DB
}

func (Migrate) Default() Migrate {
	return Migrate{DB: Observer{}.Default().DB}
}
