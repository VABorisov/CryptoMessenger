package mainscreen

import (
	"context"
	"fmt"
	"image"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"log/slog"

	"github.com/VABorisov/CryptoMessenger/internal/models"
	"github.com/VABorisov/CryptoMessenger/internal/transport/udp"
)

type EncryptionMode string

const (
	ModeBlock  EncryptionMode = "Блочный шифр (AES)"
	ModeStream EncryptionMode = "Поточный шифр (ChaCha20)"
)

type Message struct {
	Username string
	Text     string
	Image    image.Image
	IsSent   bool
}

var (
	udpHandler *udp.UDPHandler
	peerAddr   string
)

func ShowMainWindow(a fyne.App, username string, isAuthenticated bool, ctx context.Context, logger *slog.Logger) {
	w := a.NewWindow("CryptoMessenger — " + username)
	w.Resize(fyne.NewSize(1200, 800))
	w.CenterOnScreen()

	// Чат
	messagesList := container.NewVBox()
	scroll := container.NewScroll(messagesList)
	scroll.Direction = container.ScrollVerticalOnly

	// Ввод текста
	textEntry := widget.NewMultiLineEntry()
	textEntry.SetPlaceHolder("Введите сообщение...")
	textEntry.Wrapping = fyne.TextWrapWord

	// Шифрование
	encryptMode := widget.NewSelect([]string{string(ModeBlock), string(ModeStream)}, nil)
	encryptMode.SetSelected(string(ModeBlock))

	// Изображение
	var attachedImage *canvas.Image
	var attachedImagePath string
	imagePreview := container.NewCenter(widget.NewLabel("Изображение не прикреплено"))

	attachBtn := widget.NewButton("Прикрепить изображение", func() {
		if !isAuthenticated {
			dialog.ShowInformation("Доступ запрещён", "Для отправки изображений нужно авторизоваться", w)
			return
		}

		// Большой диалог выбора файла
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			img, _, err := image.Decode(reader)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			attachedImage = canvas.NewImageFromImage(img)
			attachedImage.FillMode = canvas.ImageFillContain
			attachedImage.SetMinSize(fyne.NewSize(400, 300)) // больше предпросмотр

			imagePreview.RemoveAll()
			imagePreview.Add(container.NewCenter(
				container.NewVBox(
					attachedImage,
					widget.NewLabel("✓ "+reader.URI().Name()),
				),
			))
			imagePreview.Refresh()

			attachedImagePath = reader.URI().Path()
		}, w)

		// Делаем диалог большим
		fileDialog.Resize(fyne.NewSize(1000, 650))

		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".ppm", ".jpg", ".jpeg", ".JPG", ".JPEG"}))
		fileDialog.Show()
	})

	// === UDP Настройки ===
	localPortEntry := widget.NewEntry()
	localPortEntry.SetPlaceHolder("Мой порт (пусто = автоматический)")

	setListenerBtn := widget.NewButton("Установить передатчик", nil) // временно nil, зададим ниже

	// Поле собеседника
	peerEntry := widget.NewEntry()
	peerEntry.SetPlaceHolder("IP:порт собеседника (например, 192.168.1.100:54321)")

	connectBtn := widget.NewButton("Подключиться", nil) // временно nil

	// Обработчик установки передатчика
	setListenerBtn.SetText("Установить передатчик")
	setListenerBtn.OnTapped = func() {
		portStr := strings.TrimSpace(localPortEntry.Text)
		port := 0
		if portStr != "" {
			var err error
			port, err = strconv.Atoi(portStr)
			if err != nil || port < 1 || port > 65535 {
				dialog.ShowError(fmt.Errorf("Неверный формат порта"), w)
				setListenerBtn.Importance = widget.DangerImportance
				setListenerBtn.Text = "Ошибка"
				setListenerBtn.Refresh()
				return
			}
		}

		var err error
		udpHandler, err = udp.NewUDPHandler(logger, port)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Не удалось запустить передатчик: %w", err), w)
			setListenerBtn.Importance = widget.DangerImportance
			setListenerBtn.Text = "Ошибка запуска"
			setListenerBtn.Refresh()
			return
		}

		dialog.ShowInformation("Успех", fmt.Sprintf("Передатчик запущен на порту %d", udpHandler.ListenPort()), w)
		setListenerBtn.Importance = widget.SuccessImportance
		setListenerBtn.Text = "Запущено"
		setListenerBtn.Disable()
		setListenerBtn.Refresh()

		go receiveMessages(w, messagesList, scroll)
	}

	// Enable/disable для поля порта
	setListenerBtn.Disable()
	localPortEntry.OnChanged = func(s string) {
		if strings.TrimSpace(s) != "" {
			setListenerBtn.Enable()
		} else {
			setListenerBtn.Disable()
		}
	}

	// Обработчик подключения к собеседнику
	connectBtn.SetText("Подключиться")
	connectBtn.OnTapped = func() {
		addr := strings.TrimSpace(peerEntry.Text)
		if addr == "" {
			dialog.ShowInformation("Ошибка", "Введите адрес собеседника", w)
			return
		}

		testMsg := models.Message{Username: username}
		if err := udpHandler.Send(addr, testMsg); err != nil {
			dialog.ShowError(fmt.Errorf("Не удалось подключиться: %w", err), w)
			connectBtn.Importance = widget.DangerImportance
			connectBtn.Text = "Ошибка"
			connectBtn.Refresh()
			return
		}

		peerAddr = addr
		dialog.ShowInformation("Успех", "Подключено к "+addr, w)
		connectBtn.Importance = widget.SuccessImportance
		connectBtn.Text = "Подключено"
		connectBtn.Refresh()
	}

	connectBtn.Disable()
	peerEntry.OnChanged = func(s string) {
		if strings.TrimSpace(s) != "" {
			connectBtn.Enable()
		} else {
			connectBtn.Disable()
		}
	}

	// Отправка сообщения
	sendBtn := widget.NewButton("Отправить", func() {
		if !isAuthenticated {
			dialog.ShowInformation("Доступ запрещён", "Для отправки сообщений нужно авторизоваться", w)
			return
		}

		text := textEntry.Text
		if text == "" && attachedImagePath == "" {
			dialog.ShowInformation("Ошибка", "Введите текст или прикрепите изображение", w)
			return
		}

		if udpHandler == nil {
			dialog.ShowInformation("Ошибка", "Сначала установите передатчик", w)
			return
		}

		if peerAddr == "" {
			dialog.ShowInformation("Ошибка", "Сначала подключитесь к собеседнику", w)
			return
		}

		msg := models.Message{
			Username: username,
			Text:     text,
			HasImage: attachedImagePath != "",
		}

		if err := udpHandler.Send(peerAddr, msg); err != nil {
			dialog.ShowError(fmt.Errorf("Ошибка отправки: %w", err), w)
			return
		}

		addMessage(messagesList, scroll, Message{
			Username: username,
			Text:     text,
			Image:    attachedImage.Image,
			IsSent:   true,
		})

		textEntry.SetText("")
		attachedImagePath = ""
		imagePreview.RemoveAll()
		imagePreview.Add(widget.NewLabel("Изображение не прикреплено"))
		imagePreview.Refresh()
	})

	// Блокировка элементов, если не авторизован
	if !isAuthenticated {
		textEntry.Disable()
		attachBtn.Disable()
		sendBtn.Disable()
		encryptMode.Disable()
		localPortEntry.Disable()
		setListenerBtn.Disable()
		peerEntry.Disable()
		connectBtn.Disable()
	}

	// Нижняя панель ввода
	inputBox := container.NewBorder(
		nil, nil,
		container.NewHBox(attachBtn, widget.NewLabel("Шифрование:"), encryptMode),
		container.NewHBox(sendBtn),
		textEntry,
	)

	// Панель UDP
	udpBox := container.NewVBox(
		widget.NewLabelWithStyle("UDP настройки", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(3,
			widget.NewLabel("Мой порт:"),
			container.NewMax(localPortEntry), // растягиваем поле
			setListenerBtn,
		),
		container.NewGridWithColumns(3,
			widget.NewLabel("Собеседник:"),
			container.NewMax(peerEntry), // растягиваем поле
			connectBtn,
		),
		widget.NewSeparator(),
	)

	// Основной layout
	content := container.NewBorder(
		udpBox,
		inputBox,
		nil, nil,
		container.NewVBox(
			widget.NewLabelWithStyle("Чат", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			scroll,
			widget.NewSeparator(),
			widget.NewLabel("Прикреплённое изображение:"),
			imagePreview,
		),
	)

	w.SetContent(container.NewPadded(content))
	w.SetMaster()

	w.SetOnClosed(func() {
		if udpHandler != nil {
			udpHandler.Close()
		}
	})

	w.Show()
}

func receiveMessages(w fyne.Window, list *fyne.Container, scroll *container.Scroll) {
	for incoming := range udpHandler.ReceiveChannel() {
		fyne.Do(func() {
			addMessage(list, scroll, Message{
				Username: incoming.Msg.Username,
				Text:     incoming.Msg.Text,
				Image:    nil, // пока без изображений
				IsSent:   false,
			})
		})
	}
}

func addMessage(list *fyne.Container, scroll *container.Scroll, msg Message) {
	bubble := container.NewVBox()

	if msg.IsSent {
		bubble.Add(widget.NewLabelWithStyle("Вы ("+msg.Username+")", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}))
	} else {
		bubble.Add(widget.NewLabelWithStyle(msg.Username, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	}

	if msg.Text != "" {
		bubble.Add(widget.NewLabel(msg.Text))
	}

	if msg.Image != nil {
		img := canvas.NewImageFromImage(msg.Image)
		img.FillMode = canvas.ImageFillContain
		img.SetMinSize(fyne.NewSize(300, 200))
		bubble.Add(img)
	}

	bubble.Add(widget.NewSeparator())

	if msg.IsSent {
		list.Add(container.NewHBox(container.NewMax(bubble)))
	} else {
		list.Add(container.NewHBox(bubble))
	}

	list.Refresh()
	scroll.ScrollToBottom()
}
