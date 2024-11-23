package repositories

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	Path "path/filepath"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories/directories"
	"saymow/version-manager/app/repositories/filesystems"
	"strings"

	"github.com/golang-collections/collections/set"
)

func (repository *Repository) GetStatus() *Status {
	status := Status{}
	seenPaths := set.New()
	trackedPaths := set.New()

	for _, file := range repository.dir.CollectAllFiles() {
		trackedPaths.Insert(file.Filepath)
	}

	for _, change := range repository.index {
		trackedPaths.Insert(change.GetPath())
	}

	Path.Walk(repository.fs.Root, func(filepath string, info fs.FileInfo, err error) error {
		errors.Check(err)
		if repository.fs.Root == filepath || strings.HasPrefix(filepath, Path.Join(repository.fs.Root, filesystems.REPOSITORY_FOLDER_NAME)) {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		seenPaths.Insert(filepath)

		savedFile := repository.findSavedFile(filepath)
		stagedChange := repository.findStagedChange(filepath)

		if savedFile == nil && stagedChange == nil {
			status.WorkingDir.UntrackedFilePaths = append(status.WorkingDir.UntrackedFilePaths, filepath)
			return nil
		}

		file, err := os.Open(filepath)
		errors.Check(err)
		defer errors.CheckFn(file.Close)

		var buffer bytes.Buffer
		chunkBuffer := make([]byte, 1024)

		for {
			n, err := file.Read(chunkBuffer)

			if err != nil && err != io.EOF {
				errors.Error(err.Error())
			}
			if n == 0 {
				break
			}

			_, err = buffer.Write(chunkBuffer[:n])
			errors.Check(err)
		}

		hasher := sha256.New()
		_, err = hasher.Write(buffer.Bytes())
		errors.Check(err)
		fileHash := hex.EncodeToString(hasher.Sum(nil))

		if stagedChange != nil {
			if stagedChange.ChangeType == directories.Removal {
				status.Staged.RemovedFilePaths = append(status.Staged.RemovedFilePaths, stagedChange.Removal.Filepath)
			} else if stagedChange.ChangeType == directories.Conflict {
				status.Staged.ConflictedFilesPaths = append(status.Staged.ConflictedFilesPaths, ConflictedFileStatus{
					Filepath: stagedChange.GetPath(),
					Message:  stagedChange.Conflict.Message,
				})

				if stagedChange.Conflict.ObjectName != fileHash {
					status.WorkingDir.ModifiedFilePaths = append(status.WorkingDir.ModifiedFilePaths, filepath)
				}
			} else {
				if savedFile == nil {
					status.Staged.CreatedFilesPaths = append(status.Staged.CreatedFilesPaths, filepath)
				} else {
					status.Staged.ModifiedFilePaths = append(status.Staged.ModifiedFilePaths, filepath)
				}

				if stagedChange.File.ObjectName != fileHash {
					status.WorkingDir.ModifiedFilePaths = append(status.WorkingDir.ModifiedFilePaths, filepath)
				}
			}
		} else {
			if savedFile.ObjectName != fileHash {
				status.WorkingDir.ModifiedFilePaths = append(status.WorkingDir.ModifiedFilePaths, filepath)
			}
		}

		return nil
	})

	trackedPaths.Difference(seenPaths).Do(func(i interface{}) {
		filepath := i.(string)

		if repository.findStagedChange(filepath) != nil {
			status.Staged.RemovedFilePaths = append(status.Staged.RemovedFilePaths, filepath)
		} else {
			status.WorkingDir.RemovedFilePaths = append(status.WorkingDir.RemovedFilePaths, filepath)
		}
	})

	return &status
}
