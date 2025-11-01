<h1 align="center">Vcs</h1>

My simple version control system written in go.

## Demo

https://github.com/user-attachments/assets/855a16cd-d38b-4901-bd10-6c938d6fbfa5

## Commands

```
Usage: vcs <command> [flags]

Flags:
  -h, --help    Show context-sensitive help.

Commands:
  init [flags]
    Initialize a repository in the current directory.

  add <path> ... [flags]
    Add files to the index.

  rm <path> ... [flags]
    Remove files from the index and working directory.

  save [flags]
    Create a save point with the current index.

  status [flags]
    Show the index and working directory status.

  restore <path> [flags]
    Restore files from index or file tree.

    Restore cover 2 usecases:

     1. Restore HEAD + index (...and remove the index change).

        It can be used to restore the current head + index changes. Index     
        changes have higher priorities. Initialy Restore will look for your   
        change in the index, if found, the index change is applied. Otherwise,
        Restore will apply the HEAD changes.

     2. Restore Save

        It can be used to restore existing Saves to the current working       
        directory.

    Caveats:

      - Restore will remove the existing changes in the path (forever) and
        restore reference.

      - You can use Restore to recover a deleted file from the index or from a
        Save.

      - The HEAD is not changed during Restore.

  logs [flags]
    Show the repository saves logs.

  refs [flags]
    Show the repository saves refs.

  ref [flags]
    Create a reference in the current Save point.

  load <name> [flags]
    Load the files tree to the current working directory. HEAD is updated
    accordingly with name.

  merge <name> [flags]
    Merge name files tree to the current file tree.
```
