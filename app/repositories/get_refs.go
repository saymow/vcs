package repositories

import "saymow/version-manager/app/repositories/filesystems"

func (repository *Repository) GetRefs() filesystems.Refs {
	return *repository.refs
}
