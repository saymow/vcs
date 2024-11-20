package repositories

import (
	"fmt"
	"saymow/version-manager/app/pkg/fixtures"
	"saymow/version-manager/app/repositories/directories"
	"saymow/version-manager/app/repositories/filesystems"
	"testing"

	fsAssert "gotest.tools/v3/assert"
)

func TestSaveIndex(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	// Check empty index
	{
		repository.SaveIndex()

		fileContent := fixtures.ReadFile(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.INDEX_FILE_NAME))

		fsAssert.Equal(t, fileContent, "Tracked files:\n\n")
	}

	// Check non empty index
	{
		repository.index = append(
			repository.index,
			&directories.Change{ChangeType: directories.Modification, File: &directories.File{Filepath: dir.Join("1.txt"), ObjectName: "1.txt-object"}},
			&directories.Change{ChangeType: directories.Modification, File: &directories.File{Filepath: dir.Join("a", "b", "6.txt"), ObjectName: "6.txt-object"}},
			&directories.Change{ChangeType: directories.Removal, Removal: &directories.FileRemoval{Filepath: dir.Join("a", "b", "5.txt")}},
			&directories.Change{ChangeType: directories.Modification, File: &directories.File{Filepath: dir.Join("a", "b", "7.txt"), ObjectName: "7.txt-object"}},
			&directories.Change{ChangeType: directories.Modification, File: &directories.File{Filepath: dir.Join("a", "b", "c", "8.txt"), ObjectName: "8.txt-object"}},
			&directories.Change{ChangeType: directories.Removal, Removal: &directories.FileRemoval{Filepath: dir.Join("a", "b", "c", "9.txt")}},
		)

		repository.SaveIndex()

		received := fixtures.ReadFile(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.INDEX_FILE_NAME))
		expected := `Tracked files:

%s	(modified)
1.txt-object
%s	(modified)
6.txt-object
%s	(removed)
%s	(modified)
7.txt-object
%s	(modified)
8.txt-object
%s	(removed)
`

		fsAssert.Equal(
			t,
			received,
			fmt.Sprintf(
				expected,
				dir.Join("1.txt"),
				dir.Join("a", "b", "6.txt"),
				dir.Join("a", "b", "5.txt"),
				dir.Join("a", "b", "7.txt"),
				dir.Join("a", "b", "c", "8.txt"),
				dir.Join("a", "b", "c", "9.txt"),
			),
		)

		// Check index updates
		{
			repository.index = []*directories.Change{repository.index[0], repository.index[2], repository.index[4]}

			repository.SaveIndex()

			received := fixtures.ReadFile(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.INDEX_FILE_NAME))
			expected := `Tracked files:

%s	(modified)
1.txt-object
%s	(removed)
%s	(modified)
8.txt-object
`

			fsAssert.Equal(
				t,
				received,
				fmt.Sprintf(
					expected,
					dir.Join("1.txt"),
					dir.Join("a", "b", "5.txt"),
					dir.Join("a", "b", "c", "8.txt"),
				),
			)
		}
	}
}
