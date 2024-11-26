package repositories

type Refs struct {
	Head string
	Refs map[string]string
}

func (repository *Repository) GetRefs() *Refs {
	return &Refs{
		Head: repository.head,
		Refs: *repository.refs,
	}
}
