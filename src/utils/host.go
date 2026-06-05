package utils

import (
	"os"
	"path/filepath"
)

func MakePath() string {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		Error("Nie można pobrać katalogu domowego: %v", err)
	}

	dashPath := filepath.Join(homeDir, "Documents", "Dash")

	err = os.MkdirAll(dashPath, os.ModePerm)
	if err != nil {
		Error("Nie udało się stworzyć folderu Dash: %v", err)
	}

	Info("Twój folder na pliki statyczne to: %s", dashPath)

	return dashPath
}
