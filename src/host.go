package src

import (
	"os"
	"path/filepath"

	"github.com/mslotwinski-dev/dash/src/utils"
)

func MakePath() string {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		utils.Error("Nie można pobrać katalogu domowego: %v", err)
	}

	dashPath := filepath.Join(homeDir, "Documents", "Dash")

	err = os.MkdirAll(dashPath, os.ModePerm)
	if err != nil {
		utils.Error("Nie udało się stworzyć folderu Dash: %v", err)
	}

	utils.Info("Twój folder na pliki statyczne to: %s", dashPath)

	return dashPath
}
