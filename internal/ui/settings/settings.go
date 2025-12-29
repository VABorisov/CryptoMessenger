package settings

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
	"github.com/VABorisov/CryptoMessenger/internal/ui/login"
	"github.com/VABorisov/CryptoMessenger/internal/ui/mainscreen"
)

func ShowDBSettingsWindow(a fyne.App, ctx context.Context, logger *slog.Logger) {
	w := a.NewWindow("Credentials Store")
	w.SetOnClosed(func() {
		a.Quit()
	})
	w.Resize(fyne.NewSize(350, 150))
	w.CenterOnScreen()

	addrEntry := widget.NewEntry()
	addrEntry.SetPlaceHolder("Credential Store address (e.g. localhost:6379)")

	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("Credential Store password")

	connectBtn := widget.NewButton("Connect", func() {
		addr := strings.TrimSpace(addrEntry.Text)
		pass := passEntry.Text

		authService, err := credentialsstore.NewRedisCredentialsStore(ctx, logger, addr, pass)
		if err != nil {
			dialog.ShowError(fmt.Errorf("connection error: %w", err), w)
			return
		}

		w.Hide()
		login.ShowLoginWindow(a, authService, ctx, logger)
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

	connectBtn.Disable()

	updateButton := func() {
		if strings.TrimSpace(addrEntry.Text) != "" && passEntry.Text != "" {
			connectBtn.Enable()
		} else {
			connectBtn.Disable()
		}
	}

	addrEntry.OnChanged = func(_ string) { updateButton() }
	passEntry.OnChanged = func(_ string) { updateButton() }

	form := container.NewVBox(
		widget.NewLabelWithStyle("Connect to Credentials Store", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		container.NewMax(addrEntry),
		container.NewMax(passEntry),
		container.NewCenter(
			container.NewHBox(
				container.NewPadded(connectBtn),
				container.NewPadded(guestBtn),
			),
		),
	)

	w.SetContent(container.NewPadded(form))
	w.Show()
}
