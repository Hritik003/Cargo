package detection

import (
	"errors"
	"os"
	"path/filepath"
)

var fileNames = map[string]string{"composer.json": "php", "requirements.txt": "python", "package.json": "javascript", "nuget.config": "dotnet", "pom.xml": "java"}

func (stack *Stack) Detect(directorypath string) (*Stack, error) {
	return detectLanguage(stack, directorypath)
}

func detectLanguage(s *Stack, directorypath string) (*Stack, error) {
	for key, value := range fileNames {
		fileName := filepath.Join(directorypath, key)
		_, err := os.Stat(fileName)
		if !os.IsNotExist(err) {
			s.Name = value
			break
		}

	}
	s.Version, s.Database, s.Framework, s.FrameworkVersion = "Nil", "Nil", "Nil", "Nil"
	if s.Name == "" {
		error := errors.New("language could not be detected")
		return nil, error
	}
	return s, nil
}
