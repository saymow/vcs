package repositories

import (
	"fmt"
	"saymow/version-manager/app/pkg/fixtures"
	"saymow/version-manager/app/repositories/directories"
	"saymow/version-manager/app/repositories/filesystems"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	fsAssert "gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestCreateSave(t *testing.T) {
	dir, _ := fixtureGetBaseProject(t)
	defer dir.Remove()

	// Check initial save

	indexFilepath := dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.INDEX_FILE_NAME)
	index := fmt.Sprintf(`Tracked files:
	
%s	(modified)
1.txt-object
%s	(modified)
4.txt-object
%s	(modified)
6.txt-object
`, dir.Join("1.txt"),
		dir.Join("a", "4.txt"),
		dir.Join("a", "b", "6.txt"),
	)
	fixtures.WriteFile(indexFilepath, []byte(index))

	repository := GetRepository(dir.Path())
	firstSave, _ := repository.CreateSave("first save")
	expectedFirstSaveFileContent := fmt.Sprintf(`%s

%s

Please do not edit the lines below.


Files:

%s	(modified)
%s
%s	(modified)
%s
%s	(modified)
%s
`,
		firstSave.Message,
		firstSave.CreatedAt.Format(time.Layout),
		firstSave.Changes[0].File.Filepath,
		firstSave.Changes[0].File.ObjectName,
		firstSave.Changes[1].File.Filepath,
		firstSave.Changes[1].File.ObjectName,
		firstSave.Changes[2].File.Filepath,
		firstSave.Changes[2].File.ObjectName,
	)

	assert.Equal(t, firstSave.Message, "first save")
	assert.Equal(t, firstSave.Parent, "")
	assert.EqualValues(
		t,
		firstSave.Changes,
		[]*directories.Change{
			{
				ChangeType: directories.Modification,
				File: &directories.File{
					Filepath:   dir.Join("1.txt"),
					ObjectName: "1.txt-object",
				},
			},
			{
				ChangeType: directories.Modification,
				File: &directories.File{
					Filepath:   dir.Join("a", "4.txt"),
					ObjectName: "4.txt-object",
				},
			},
			{
				ChangeType: directories.Modification,
				File: &directories.File{
					Filepath:   dir.Join("a", "b", "6.txt"),
					ObjectName: "6.txt-object",
				},
			},
		},
	)
	fsAssert.Assert(
		t,
		fs.Equal(
			dir.Join(filesystems.REPOSITORY_FOLDER_NAME),
			fs.Expected(
				t,
				fs.WithFile(filesystems.REFS_FILE_NAME, fmt.Sprintf("Refs:\n\n%s\n%s\n", filesystems.INITIAL_REF_NAME, firstSave.Id)),
				fs.WithFile(filesystems.HEAD_FILE_NAME, filesystems.INITIAL_REF_NAME),
				fs.WithFile(filesystems.INDEX_FILE_NAME, "Tracked files:\n\n"),
				fs.WithDir(filesystems.SAVES_FOLDER_NAME,
					fs.WithFile(firstSave.Id, expectedFirstSaveFileContent),
				),
				fs.WithDir(filesystems.OBJECTS_FOLDER_NAME),
			),
		),
	)

	// Check second save

	index = fmt.Sprintf(`Tracked files:
	
%s	(removed)
%s	(removed)
%s	(modified)
8.txt-object
`, dir.Join("1.txt"),
		dir.Join("a", "4.txt"),
		dir.Join("a", "b", "c", "8.txt"),
	)
	fixtures.WriteFile(indexFilepath, []byte(index))

	repository = GetRepository(dir.Path())
	secondSave, _ := repository.CreateSave("second save")
	expectedSecondSaveFileContent := fmt.Sprintf(`%s
%s
%s

Please do not edit the lines below.


Files:

%s	(removed)
%s	(removed)
%s	(modified)
%s
`,
		secondSave.Message,
		secondSave.Parent,
		secondSave.CreatedAt.Format(time.Layout),
		secondSave.Changes[0].Removal.Filepath,
		secondSave.Changes[1].Removal.Filepath,
		secondSave.Changes[2].File.Filepath,
		secondSave.Changes[2].File.ObjectName,
	)

	assert.Equal(t, secondSave.Message, "second save")
	assert.Equal(t, secondSave.Parent, firstSave.Id)
	assert.EqualValues(
		t,
		secondSave.Changes,
		[]*directories.Change{
			{
				ChangeType: directories.Removal,
				Removal:    &directories.FileRemoval{Filepath: dir.Join("1.txt")},
			},
			{
				ChangeType: directories.Removal,
				Removal:    &directories.FileRemoval{Filepath: dir.Join("a", "4.txt")},
			},
			{
				ChangeType: directories.Modification,
				File: &directories.File{
					Filepath:   dir.Join("a", "b", "c", "8.txt"),
					ObjectName: "8.txt-object",
				},
			},
		},
	)
	fsAssert.Assert(
		t,
		fs.Equal(
			dir.Join(filesystems.REPOSITORY_FOLDER_NAME),
			fs.Expected(
				t,
				fs.WithFile(filesystems.REFS_FILE_NAME, fmt.Sprintf("Refs:\n\n%s\n%s\n", filesystems.INITIAL_REF_NAME, secondSave.Id)),
				fs.WithFile(filesystems.HEAD_FILE_NAME, filesystems.INITIAL_REF_NAME),
				fs.WithFile(filesystems.INDEX_FILE_NAME, "Tracked files:\n\n"),
				fs.WithDir(filesystems.SAVES_FOLDER_NAME,
					fs.WithFile(firstSave.Id, expectedFirstSaveFileContent),
					fs.WithFile(secondSave.Id, expectedSecondSaveFileContent),
				),
				fs.WithDir(filesystems.OBJECTS_FOLDER_NAME),
			),
		),
	)
}
