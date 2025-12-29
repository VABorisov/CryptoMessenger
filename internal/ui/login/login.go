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

	"github.com/VABorisov/CryptoMessenger/internal/credentialsstore"
	"github.com/VABorisov/CryptoMessenger/internal/crypto/asymmetric"
	"github.com/VABorisov/CryptoMessenger/internal/ui/mainscreen"
)

func ShowLoginWindow(a fyne.App, authService credentialsstore.CredentialsStore, ctx context.Context, logger *slog.Logger) {
	w := a.NewWindow("Authentication")
	w.SetOnClosed(func() {
		a.Quit()
	})
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

		rsaCipher, err := asymmetric.NewRSAAsymmetricCipher(logger, username)
		if err != nil {
			logger.Error("failed to initialize RSA keys for new user", "username", username, "error", err)
			dialog.ShowError(fmt.Errorf("Failed to initialize RSA keys: %w", err), w)
			return
		}

		dialog.ShowInformation("Success", "User successfully created!", w)
		usernameEntry.SetText("")
		passwordEntry.SetText("")
		usernameEntry.FocusGained()
		_ = rsaCipher
	})

	signInBtn := widget.NewButton("Sign in", func() {
		username := strings.TrimSpace(usernameEntry.Text)
		password := passwordEntry.Text

		ok, err := authService.Authenticate(ctx, username, password)
		if err != nil {
			if strings.Contains(err.Error(), "user not found") {
				dialog.ShowInformation("Error", "Invalid username or password", w)
			}
			dialog.ShowError(fmt.Errorf("User authentication error: %w", err), w)
			return
		}
		if !ok {
			dialog.ShowInformation("Error", "Invalid username or password", w)
			return
		}

		rsaCipher, err := asymmetric.NewRSAAsymmetricCipher(logger, username)
		if err != nil {
			logger.Error("failed to initialize RSA keys for user", "username", username, "error", err)
			dialog.ShowError(fmt.Errorf("Failed to initialize RSA keys: %w", err), w)
			return
		}

		w.Hide()
		mainscreen.ShowMainWindow(a, usernameEntry.Text, true, ctx, logger, rsaCipher)
	})

	guestBtn := widget.NewButton("Enter as Guest", func() {
		rsaCipher, err := asymmetric.NewRSAAsymmetricCipher(logger, "Guest")
		if err != nil {
			logger.Error("failed to generate guest RSA keys", "error", err)
			dialog.ShowError(fmt.Errorf("Failed to generate temporary keys: %w", err), w)
			return
		}

		w.Hide()
		mainscreen.ShowMainWindow(a, "Guest", false, ctx, logger, rsaCipher)
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
