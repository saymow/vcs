package fixtures

import (
	"io"
	"os"
	"saymow/version-manager/app/pkg/errors"
	"strings"
)

func FileExists(filepath string) bool {
	f, err := os.Open(filepath)
	if err != nil {
		defer f.Close()
	}

	return !os.IsNotExist(err)
}

func ReadFile(filepath string) string {
	file, err := os.Open(filepath)
	errors.Check(err)
	defer file.Close()

	var str strings.Builder
	buffer := make([]byte, 256)

	for {
		n, err := file.Read(buffer)

		if err != nil && err != io.EOF {
			errors.Error(err.Error())
		}
		if n == 0 {
			break
		}

		_, err = str.WriteString(string(buffer[:n]))
		errors.Check(err)
	}

	return str.String()
}

func WriteFile(filepath string, content []byte) {
	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		if !os.IsNotExist(err) {
			errors.Error(err.Error())
		}

		file, err = os.Create(filepath)
		errors.Check(err)
	}

	defer file.Close()

	_, err = file.Write(content)
	errors.Check(err)
}

func MakeDirs(filepaths ...string) {
	for _, filepath := range filepaths {
		err := os.Mkdir(filepath, 0644)
		errors.Check(err)
	}
}

func RemoveFile(filepath string) {
	err := os.Remove(filepath)
	errors.Check(err)
}
