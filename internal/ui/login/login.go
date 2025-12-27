package login

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/VABorisov/CryptoMessenger/internal/auth"
	"github.com/VABorisov/CryptoMessenger/internal/ui/mainscreen"
)

func ShowLoginWindow(a fyne.App, authService auth.AuthService, ctx context.Context, logger *slog.Logger) {
	w := a.NewWindow("Authentication")
	w.Resize(fyne.NewSize(300, 150))
	w.CenterOnScreen()

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Username")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	signUpBtn := widget.NewButton("Sign up", func() {
		username := strings.TrimSpace(usernameEntry.Text)
		password := passwordEntry.Text

		err := authService.CreateUser(ctx, username, password)
		if err != nil {
			if strings.Contains(err.Error(), "user not found") {
				dialog.ShowInformation("Error", "User already exists", w)
			}
			dialog.ShowError(fmt.Errorf("User creation error: %w", err), w)
			return
		}

		dialog.ShowInformation("Success", "User successfully created!", w)
		usernameEntry.SetText("")
		passwordEntry.SetText("")
		usernameEntry.FocusGained()
	})

	signInBtn := widget.NewButton("Sign in", func() {
		username := strings.TrimSpace(usernameEntry.Text)
		password := passwordEntry.Text

		ok, err := authService.Authenticate(ctx, username, password)
		if err != nil {
			if strings.Contains(err.Error(), "user not found") {
				dialog.ShowInformation("Error", "nvalid username or password", w)
			}
			dialog.ShowError(fmt.Errorf("User authentication error: %w", err), w)
			return
		}

		if !ok {
			dialog.ShowInformation("Error", "Invalid username or password", w)
			return
		}

		w.Close()
		mainscreen.ShowMainWindow(a, usernameEntry.Text, true, ctx, logger)
	})

	guestBtn := widget.NewButton("Entry as Guest", func() {
		w.Close()
		mainscreen.ShowMainWindow(a, "Guest", false, ctx, logger)
	})

	signUpBtn.Disable()
	signInBtn.Disable()

	updateButtons := func() {
		usernameFilled := strings.TrimSpace(usernameEntry.Text) != ""
		passwordFilled := passwordEntry.Text != ""

		if usernameFilled && passwordFilled {
			signUpBtn.Enable()
			signInBtn.Enable()
		} else {
			signUpBtn.Disable()
			signInBtn.Disable()
		}
	}

	usernameEntry.OnChanged = func(_ string) { updateButtons() }
	passwordEntry.OnChanged = func(_ string) { updateButtons() }

	content := container.NewVBox(
		widget.NewLabelWithStyle("Login", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		container.NewMax(usernameEntry),
		container.NewMax(passwordEntry),
		container.NewCenter(
			container.NewHBox(
				container.NewPadded(signUpBtn),
				container.NewPadded(signInBtn),
				container.NewPadded(guestBtn),
			),
		),
	)

	w.SetContent(container.NewPadded(content))
	w.Show()

	usernameEntry.FocusGained()
}
