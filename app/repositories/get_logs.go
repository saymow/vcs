package repositories

import (
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/repositories/filesystems"
	"slices"
)

type Log struct {
	Head    string
	History []*SaveLog
}

func (repository *Repository) GetLogs() *Log {
	save := repository.getSave(repository.head)

	if save == nil {
		// repostory without saves history

		return &Log{
			Head:    repository.head,
			History: []*SaveLog{},
		}
	}

	savesToRefsMap := collections.InvertMap(*repository.refs)

	// By default the save checkpoints is ordered by createdAt in ascending order.
	// The other way around is better for logging.
	slices.Reverse(save.Checkpoints)

	return &Log{
		Head: repository.head,
		History: collections.Map(save.Checkpoints, func(checkpoint *filesystems.Checkpoint, _ int) *SaveLog {
			var refs []string

			if mapSaves, ok := savesToRefsMap[checkpoint.Id]; ok {
				refs = mapSaves
			}

			return &SaveLog{Checkpoint: checkpoint, Refs: refs}
		}),
	}
}
