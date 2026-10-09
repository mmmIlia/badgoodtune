package domain

// ID is the canonical textual identifier used by inner layers.
// UUID parsing and database representations belong to boundary adapters.
type ID string

func (id ID) String() string {
	return string(id)
}

func (id ID) IsZero() bool {
	return id == ""
}
