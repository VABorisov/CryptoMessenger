package main

import (
	"context"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/VABorisov/CryptoMessenger/internal/logger"
	"github.com/VABorisov/CryptoMessenger/internal/ui/settings"
)

func main() {
	logger := logger.NewJSONLogger()
	myApp := app.NewWithID("com.vaborisov.cryptomessenger")
	myApp.SetIcon(loadIcon())

	settings.ShowDBSettingsWindow(myApp, context.TODO(), logger)

	myApp.Run()
}

func loadIcon() fyne.Resource {
	f, err := os.ReadFile("appicon.png")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("icon.png", f)
}
