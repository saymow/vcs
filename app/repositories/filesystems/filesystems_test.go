package filesystems

import (
	"saymow/version-manager/app/repositories/directories"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSaveContains(t *testing.T) {
	s0 := &Checkpoint{
		Id:        "s0",
		Parent:    "",
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}
	s1 := &Checkpoint{
		Id:        "s1",
		Parent:    "s0",
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}
	s2 := &Checkpoint{
		Id:        "s2",
		Parent:    "s1",
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}
	s3 := &Checkpoint{
		Id:        "s3",
		Parent:    "s2",
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}

	save := &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, s2, s3},
	}
	assert.True(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, s2, s3},
	}
	assert.True(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, s2, s3},
	}
	assert.True(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, s2},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, s2, s3},
	}
	assert.True(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, s2, s3},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, s2},
	}
	assert.True(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, s2},
	}
	assert.True(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, s2},
	}
	assert.True(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, s2},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1},
	}
	assert.True(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1},
	}
	assert.True(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0},
	}
	assert.True(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0},
			},
		),
	)

	// False

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0},
	}
	assert.False(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, s2, s3},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0},
	}
	assert.False(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, s2},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0},
	}
	assert.False(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1},
	}
	assert.False(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, s2, s3},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1},
	}
	assert.False(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, s2},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, s2},
	}
	assert.False(
		t,
		save.Contains(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, s2, s3},
			},
		),
	)

}

func TestSaveFindFirstCommonParent(t *testing.T) {
	s0 := &Checkpoint{
		Id:        "s0",
		Parent:    "",
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}
	s1 := &Checkpoint{
		Id:        "s1",
		Parent:    "s0",
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}

	// branch a

	as2 := &Checkpoint{
		Id:        "as2",
		Parent:    "s1",
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}
	as3 := &Checkpoint{
		Id:        "as3",
		Parent:    "as2",
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}
	as4 := &Checkpoint{
		Id:        "as4",
		Parent:    "as3",
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}

	// branch b

	bs2 := &Checkpoint{
		Id:        "bs2",
		Parent:    "s1",
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}
	bs3 := &Checkpoint{
		Id:        "bs3",
		Parent:    "bs2",
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}

	save := &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, as2, as3, as4},
	}
	assert.Equal(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, bs2, bs3},
			},
		),
		s1,
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, bs2, bs3},
	}
	assert.Equal(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, as2, as3, as4},
			},
		),
		s1,
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, as2, as3},
	}
	assert.Equal(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, bs2, bs3},
			},
		),
		s1,
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, bs2, bs3},
	}
	assert.Equal(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, as2, as3},
			},
		),
		s1,
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, as2, as3, as4},
	}
	assert.Equal(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, bs2},
			},
		),
		s1,
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, bs2},
	}
	assert.Equal(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, as2, as3, as4},
			},
		),
		s1,
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, bs2, bs3},
	}
	assert.Equal(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, as2, as3, as4},
			},
		),
		s0,
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, as2, as3, as4},
	}
	assert.Equal(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, bs2, bs3},
			},
		),
		s0,
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1, as2, as3, as4},
	}
	assert.Equal(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{as2, as3},
			},
		),
		as2,
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{as2, as3},
	}
	assert.Equal(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1, as2, as3, as4},
			},
		),
		as2,
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1},
	}
	assert.Equal(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1},
			},
		),
		s0,
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0},
	}
	assert.Nil(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0, s1},
			},
		),
	)

	save = &Save{
		Id:          "",
		Checkpoints: []*Checkpoint{s0, s1},
	}

	assert.Nil(
		t,
		save.FindFirstCommonCheckpointParent(
			&Save{
				Id:          "",
				Checkpoints: []*Checkpoint{s0},
			},
		),
	)
}
