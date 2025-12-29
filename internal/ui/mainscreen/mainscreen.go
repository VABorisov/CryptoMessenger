package mainscreen

import (
	"context"
	"fmt"
	"image"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/VABorisov/CryptoMessenger/internal/crypto/asymmetric"
	"github.com/VABorisov/CryptoMessenger/internal/crypto/block"
	"github.com/VABorisov/CryptoMessenger/internal/crypto/stream"
	"github.com/VABorisov/CryptoMessenger/internal/image/formats/ppm"
	"github.com/VABorisov/CryptoMessenger/internal/models"
	"github.com/VABorisov/CryptoMessenger/internal/transport"
	"github.com/VABorisov/CryptoMessenger/internal/transport/udp"
)

type (
	Message struct {
		Username string
		Text     string
		Image    fyne.CanvasObject
		IsSent   bool
	}

	KeyExchangeState struct {
		mu                sync.Mutex
		username          string
		rsaCipher         asymmetric.AsymmetricCipher
		blockCipher       block.BlockCipher
		streamCipher      stream.StreamCipher
		localWantsConnect bool
		peerWantsConnect  bool
		connected         bool
		isInitiator       bool
		peerPublicKey     []byte
		publicTicker      *time.Ticker
		keysReceived      bool
		pingReceived      bool
	}

	EncryptionMode string
)

const (
	ModeBlock  EncryptionMode = "Block Cipher (TEA)"
	ModeStream EncryptionMode = "Stream Cipher (ChaCha20)"
)

var (
	udpTransport    transport.Transport
	peerAddr        string
	keyState        *KeyExchangeState
	connectBtn      *widget.Button
	currentPPMImage image.Image
	currentPPMData  []byte
)

func ShowMainWindow(a fyne.App, username string, isAuthenticated bool, ctx context.Context, logger *slog.Logger, rsaCipher asymmetric.AsymmetricCipher) {
	w := a.NewWindow("CryptoMessenger — " + username)
	w.SetOnClosed(func() {
		a.Quit()
	})
	w.Resize(fyne.NewSize(1400, 900))
	w.CenterOnScreen()

	keyState = &KeyExchangeState{
		username:  username,
		rsaCipher: rsaCipher,
	}

	messagesList := container.NewVBox()
	scroll := container.NewScroll(messagesList)
	scroll.Direction = container.ScrollVerticalOnly

	textEntry := widget.NewMultiLineEntry()
	textEntry.SetPlaceHolder("Enter message...")
	textEntry.Wrapping = fyne.TextWrapWord

	encryptMode := widget.NewSelect([]string{string(ModeBlock), string(ModeStream)}, nil)
	encryptMode.SetSelected(string(ModeBlock))

	var attachedImage *canvas.Image
	imagePreview := container.NewCenter(widget.NewLabel("No image attached"))

	attachBtn := widget.NewButton("Attach Image", func() {
		if !isAuthenticated {
			dialog.ShowInformation("Access Denied", "You need to authenticate to send images", w)
			return
		}

		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			data, err := io.ReadAll(reader)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Error reading file: %w", err), w)
				return
			}

			img, err := ppm.Decode(strings.NewReader(string(data)))
			if err != nil {
				dialog.ShowError(fmt.Errorf("Error decoding PPM: %w", err), w)
				return
			}

			attachedImage = canvas.NewImageFromImage(img)
			attachedImage.FillMode = canvas.ImageFillContain
			attachedImage.SetMinSize(fyne.NewSize(500, 400))

			imagePreview.RemoveAll()
			imagePreview.Add(container.NewCenter(
				container.NewVBox(
					attachedImage,
					widget.NewLabel("✓ "+reader.URI().Name()),
				),
			))
			imagePreview.Refresh()

			currentPPMImage = img
			currentPPMData = data
		}, w)

		fd.Resize(fyne.NewSize(1000, 650))
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".ppm"}))
		fd.Show()
	})

	localPortEntry := widget.NewEntry()
	localPortEntry.SetPlaceHolder("My port (empty = automatic)")

	setListenerBtn := widget.NewButton("Start Listener", nil)

	peerEntry := widget.NewEntry()
	peerEntry.SetPlaceHolder("Peer IP:port (e.g., 192.168.1.100:54321)")

	connectBtn = widget.NewButton("Connect", nil)

	setListenerBtn.SetText("Start Listener")
	setListenerBtn.OnTapped = func() {
		portStr := strings.TrimSpace(localPortEntry.Text)
		port := 0
		if portStr != "" {
			var err error
			port, err = strconv.Atoi(portStr)
			if err != nil || port < 1 || port > 65535 {
				dialog.ShowError(fmt.Errorf("Invalid port format"), w)
				setListenerBtn.Importance = widget.DangerImportance
				setListenerBtn.Text = "Error"
				setListenerBtn.Refresh()
				return
			}
		}

		var err error
		udpTransport, err = udp.NewUDPTransport(logger, port)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to start listener: %w", err), w)
			setListenerBtn.Importance = widget.DangerImportance
			setListenerBtn.Text = "Start Error"
			setListenerBtn.Refresh()
			return
		}

		dialog.ShowInformation("Success", fmt.Sprintf("Listener started on port %d", udpTransport.ListenPort()), w)
		setListenerBtn.Importance = widget.SuccessImportance
		setListenerBtn.Text = "Running"
		setListenerBtn.Disable()
		setListenerBtn.Refresh()

		go receiveMessages(logger, messagesList, scroll)
	}

	setListenerBtn.Disable()
	localPortEntry.OnChanged = func(s string) {
		if strings.TrimSpace(s) != "" {
			setListenerBtn.Enable()
		} else {
			setListenerBtn.Disable()
		}
	}

	connectBtn.OnTapped = func() {
		keyState.mu.Lock()
		connected := keyState.connected
		localWantsConnect := keyState.localWantsConnect
		currentAddr := peerAddr
		keyState.mu.Unlock()

		if connected {
			abortConnection()
			return
		}

		addr := strings.TrimSpace(peerEntry.Text)
		if err := validateAddress(addr); err != nil {
			dialog.ShowError(fmt.Errorf("Invalid address: %w", err), w)
			return
		}

		if localWantsConnect && addr != currentAddr {
			abortConnection()
			time.Sleep(100 * time.Millisecond)
		}

		if localWantsConnect && addr == currentAddr {
			abortConnection()
			return
		}

		keyState.mu.Lock()
		keyState.localWantsConnect = true
		keyState.isInitiator = true
		keyState.mu.Unlock()

		startConnection(addr)
		updateConnectButton()
	}

	sendBtn := widget.NewButton("Send", func() {
		if !isAuthenticated {
			dialog.ShowInformation("Access Denied", "Authentication required to send messages", w)
			return
		}

		text := strings.TrimSpace(textEntry.Text)
		hasText := text != ""
		hasImage := currentPPMImage != nil

		if !hasText && !hasImage {
			dialog.ShowInformation("Error", "Enter text or attach an image", w)
			return
		}

		if udpTransport == nil {
			dialog.ShowInformation("Error", "Start listener first", w)
			return
		}

		if peerAddr == "" || !keyState.connected {
			dialog.ShowInformation("Error", "Connect to peer first", w)
			return
		}

		selectedMode := encryptMode.Selected
		var encryptFunc func([]byte) ([]byte, error)
		var modeStr string

		keyState.mu.Lock()
		if selectedMode == string(ModeBlock) {
			if keyState.blockCipher == nil {
				keyState.mu.Unlock()
				dialog.ShowError(fmt.Errorf("Block cipher not initialized"), w)
				return
			}
			encryptFunc = keyState.blockCipher.Encrypt
			modeStr = "tea"
		} else {
			if keyState.streamCipher == nil {
				keyState.mu.Unlock()
				dialog.ShowError(fmt.Errorf("Stream cipher not initialized"), w)
				return
			}
			encryptFunc = keyState.streamCipher.Encrypt
			modeStr = "chacha20"
		}
		keyState.mu.Unlock()

		var dataToEncrypt []byte
		if hasText {
			dataToEncrypt = []byte(text)
		}
		if hasImage {
			if hasText {
				dataToEncrypt = append(dataToEncrypt, []byte("\n---CRYPTO_MESSENGER_PPM---\n")...)
				dataToEncrypt = append(dataToEncrypt, currentPPMData...)
			} else {
				dataToEncrypt = currentPPMData
			}
		}

		encrypted, err := encryptFunc(dataToEncrypt)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Encryption error: %w", err), w)
			return
		}

		msg := models.Message{
			Type:           models.MessageTypeRegular,
			Username:       username,
			EncryptedData:  encrypted,
			HasImage:       hasImage,
			EncryptionMode: modeStr,
		}

		if err := udpTransport.Send(peerAddr, msg); err != nil {
			dialog.ShowError(fmt.Errorf("Send error: %w", err), w)
			return
		}

		var display fyne.CanvasObject
		if hasImage {
			display = attachedImage
		}

		addMessage(messagesList, scroll, Message{
			Username: username,
			Text:     text,
			Image:    display,
			IsSent:   true,
		})

		textEntry.SetText("")
		currentPPMImage = nil
		currentPPMData = nil
		imagePreview.RemoveAll()
		imagePreview.Add(container.NewCenter(widget.NewLabel("No image attached")))
		imagePreview.Refresh()
	})

	if !isAuthenticated {
		textEntry.Disable()
		attachBtn.Disable()
		sendBtn.Disable()
		encryptMode.Disable()
	}

	inputBox := container.NewBorder(
		nil, nil,
		container.NewHBox(attachBtn, widget.NewLabel("Encryption:"), encryptMode),
		container.NewHBox(sendBtn),
		textEntry,
	)

	udpBox := container.NewVBox(
		widget.NewLabelWithStyle("Connection settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(3,
			widget.NewLabel("My port:"),
			container.NewMax(localPortEntry),
			setListenerBtn,
		),
		container.NewGridWithColumns(3,
			widget.NewLabel("Peer:"),
			container.NewMax(peerEntry),
			connectBtn,
		),
		widget.NewSeparator(),
	)

	imageSection := container.NewVBox(
		widget.NewLabelWithStyle("Attached image:", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		imagePreview,
	)

	imageSectionLimited := container.NewMax(container.NewBorder(nil, nil, nil, nil, imageSection))

	chatSection := container.NewBorder(
		widget.NewLabelWithStyle("Chat", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		scroll,
	)

	mainContent := container.NewHSplit(imageSectionLimited, chatSection)
	mainContent.SetOffset(0.5)

	content := container.NewBorder(
		udpBox,
		inputBox,
		nil, nil,
		mainContent,
	)

	w.SetContent(container.NewPadded(content))
	w.SetMaster()

	w.SetOnClosed(func() {
		if udpTransport != nil {
			udpTransport.Close()
		}
		if !isAuthenticated && keyState != nil && keyState.rsaCipher != nil {
			if err := keyState.rsaCipher.DeleteGuestKeys(); err != nil {
				logger.Error("failed to delete guest keys", "error", err)
			} else {
				logger.Info("guest keys deleted successfully")
			}
		}
	})

	updateConnectButton()
	w.Show()
}

func receiveMessages(logger *slog.Logger, list *fyne.Container, scroll *container.Scroll) {
	for in := range udpTransport.ReceiveChannel() {
		switch in.Msg.Type {
		case models.MessageTypePublicKey:
			handlePublicKey(logger, in.Msg, in.From)
		case models.MessageTypeSymmetricKey:
			handleSymmetricKey(logger, in.Msg)
		case models.MessageTypeKeyAck:
			handleKeyAck()
		case models.MessagePing:
			handlePing(logger, in.From)
		case models.MessageTypeRegular:
			handleRegularMessage(logger, in.Msg, list, scroll)
		default:
			if in.Msg.Type == "" && len(in.Msg.EncryptedData) > 0 {
				handleRegularMessage(logger, in.Msg, list, scroll)
			}
		}
	}
}

func updateConnectButton() {
	if connectBtn == nil {
		return
	}

	keyState.mu.Lock()
	local := keyState.localWantsConnect
	connected := keyState.connected
	keyState.mu.Unlock()

	fyne.Do(func() {
		switch {
		case connected:
			connectBtn.SetText("Connected")
			connectBtn.Enable()
			connectBtn.Importance = widget.SuccessImportance
		case local:
			connectBtn.SetText("Connecting...")
			connectBtn.Enable()
			connectBtn.Importance = widget.HighImportance
		default:
			connectBtn.SetText("Connect")
			connectBtn.Enable()
			connectBtn.Importance = widget.MediumImportance
		}
		connectBtn.Refresh()
	})
}

func validateAddress(addr string) error {
	if addr == "" {
		return fmt.Errorf("address cannot be empty")
	}

	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("address must be in format 'host:port' or 'IP:port' (e.g., 192.168.1.100:54321)")
	}

	host := strings.TrimSpace(parts[0])
	portStr := strings.TrimSpace(parts[1])

	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	if portStr == "" {
		return fmt.Errorf("port cannot be empty")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("port must be a number")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

func startConnection(addr string) {
	if addr == "" || udpTransport == nil {
		return
	}

	peerAddr = addr

	keyState.mu.Lock()
	keyState.keysReceived = false
	keyState.pingReceived = false
	if keyState.publicTicker != nil {
		keyState.publicTicker.Stop()
		keyState.publicTicker = nil
	}
	keyState.mu.Unlock()

	sendPublicKey(addr)

	keyState.mu.Lock()
	keyState.publicTicker = time.NewTicker(2 * time.Second)
	keyState.mu.Unlock()

	go func() {
		ticker := keyState.publicTicker
		if ticker == nil {
			return
		}

		for range ticker.C {
			keyState.mu.Lock()
			shouldStop := keyState.connected || !keyState.localWantsConnect || !keyState.isInitiator
			currentAddr := peerAddr
			keyState.mu.Unlock()

			if shouldStop {
				keyState.mu.Lock()
				if keyState.publicTicker != nil {
					keyState.publicTicker.Stop()
					keyState.publicTicker = nil
				}
				keyState.mu.Unlock()
				return
			}

			if currentAddr != "" {
				sendPublicKey(currentAddr)
			}
		}
	}()
}

func sendPublicKey(addr string) {
	if udpTransport == nil {
		return
	}

	pub, err := keyState.rsaCipher.GetPublicKey()
	if err != nil {
		return
	}

	msg := models.Message{
		Type:          models.MessageTypePublicKey,
		PublicKeyData: pub,
	}

	udpTransport.Send(addr, msg)
}

func abortConnection() {
	keyState.mu.Lock()
	keyState.localWantsConnect = false
	keyState.peerWantsConnect = false
	keyState.connected = false
	keyState.isInitiator = false
	keyState.keysReceived = false
	keyState.pingReceived = false
	if keyState.publicTicker != nil {
		keyState.publicTicker.Stop()
		keyState.publicTicker = nil
	}
	keyState.mu.Unlock()

	peerAddr = ""
	updateConnectButton()
}

func handlePublicKey(logger *slog.Logger, msg models.Message, from string) {
	keyState.mu.Lock()
	keyState.peerWantsConnect = true
	keyState.peerPublicKey = msg.PublicKeyData
	if peerAddr == "" {
		peerAddr = from
	}
	initiator := keyState.isInitiator && keyState.localWantsConnect
	keyState.mu.Unlock()

	if initiator {
		keyState.mu.Lock()
		if keyState.publicTicker != nil {
			keyState.publicTicker.Stop()
			keyState.publicTicker = nil
		}
		keyState.mu.Unlock()

		if len(keyState.peerPublicKey) > 0 {
			sendSymmetricKeys(logger)
		}
	} else {
		pub, err := keyState.rsaCipher.GetPublicKey()
		if err != nil {
			logger.Error("failed to get public key for response", "error", err)
			return
		}

		responseMsg := models.Message{
			Type:          models.MessageTypePublicKey,
			PublicKeyData: pub,
		}

		udpTransport.Send(from, responseMsg)
	}
}

func handleSymmetricKey(logger *slog.Logger, msg models.Message) {
	data, err := keyState.rsaCipher.Decrypt(msg.SymmetricKeyData)
	if err != nil {
		logger.Error("failed to decrypt symmetric keys", "error", err)
		return
	}

	if len(data) != 72 {
		logger.Error("invalid symmetric key data length", "got", len(data), "expected", 72)
		return
	}

	teaKey := data[0:16]
	chachaKey := data[16:48]
	chachaNonce := data[48:72]

	keyState.mu.Lock()
	keyState.blockCipher = block.NewTEABlockCipherWithKey(logger, teaKey)
	keyState.streamCipher = stream.NewChaCha20StreamCipherWithKey(logger, chachaKey, chachaNonce)
	keyState.keysReceived = true
	localWantsConnect := keyState.localWantsConnect
	keyState.mu.Unlock()

	udpTransport.Send(peerAddr, models.Message{Type: models.MessageTypeKeyAck})

	if localWantsConnect && peerAddr != "" {
		udpTransport.Send(peerAddr, models.Message{Type: models.MessagePing})
	}

	keyState.mu.Lock()
	if localWantsConnect && keyState.pingReceived {
		keyState.connected = true
	}
	keyState.mu.Unlock()

	updateConnectButton()
}

func handleKeyAck() {
	keyState.mu.Lock()
	keyState.keysReceived = true
	localWantsConnect := keyState.localWantsConnect
	keyState.mu.Unlock()

	if localWantsConnect && peerAddr != "" {
		udpTransport.Send(peerAddr, models.Message{Type: models.MessagePing})
	}

	keyState.mu.Lock()
	if localWantsConnect && keyState.pingReceived {
		keyState.connected = true
	}
	keyState.mu.Unlock()

	updateConnectButton()
}

func handlePing(logger *slog.Logger, from string) {
	keyState.mu.Lock()
	alreadyConnected := keyState.connected
	localWantsConnect := keyState.localWantsConnect
	keysReceived := keyState.keysReceived
	keyState.mu.Unlock()

	if !alreadyConnected {
		udpTransport.Send(from, models.Message{Type: models.MessagePing})
	}

	keyState.mu.Lock()
	keyState.pingReceived = true
	keyState.mu.Unlock()

	if localWantsConnect && keysReceived && !alreadyConnected {
		keyState.mu.Lock()
		keyState.connected = true
		keyState.mu.Unlock()
		updateConnectButton()
	}
}

func sendSymmetricKeys(logger *slog.Logger) {
	keyState.mu.Lock()
	peerPubKey := keyState.peerPublicKey
	keyState.mu.Unlock()

	if len(peerPubKey) == 0 {
		logger.Error("cannot send symmetric keys: peer public key not received")
		return
	}

	tea := block.NewTEABlockCipher(logger)
	chacha := stream.NewChaCha20StreamCipher(logger)

	keyState.mu.Lock()
	keyState.blockCipher = tea
	keyState.streamCipher = chacha
	keyState.keysReceived = true
	keyState.mu.Unlock()

	teaKey := tea.GetKey()
	chachaKey, chachaNonce := chacha.GetKey()

	payload := make([]byte, 0, 72)
	payload = append(payload, teaKey...)
	payload = append(payload, chachaKey...)
	payload = append(payload, chachaNonce...)

	encrypted, err := keyState.rsaCipher.EncryptWithPeerKey(peerPubKey, payload)
	if err != nil {
		logger.Error("failed to encrypt symmetric keys", "error", err)
		return
	}

	msg := models.Message{
		Type:             models.MessageTypeSymmetricKey,
		SymmetricKeyData: encrypted,
	}

	udpTransport.Send(peerAddr, msg)

	keyState.mu.Lock()
	localWantsConnect := keyState.localWantsConnect
	keyState.mu.Unlock()

	if localWantsConnect && peerAddr != "" {
		udpTransport.Send(peerAddr, models.Message{Type: models.MessagePing})
	}
}

func handleRegularMessage(logger *slog.Logger, msg models.Message, list *fyne.Container, scroll *container.Scroll) {
	if len(msg.EncryptedData) == 0 {
		return
	}

	keyState.mu.Lock()
	blockCipher := keyState.blockCipher
	streamCipher := keyState.streamCipher
	keyState.mu.Unlock()

	if blockCipher == nil || streamCipher == nil {
		logger.Warn("ciphers not ready for decryption")
		return
	}

	var decryptFunc func([]byte) ([]byte, error)

	switch msg.EncryptionMode {
	case "tea":
		decryptFunc = blockCipher.Decrypt
	case "chacha20":
		decryptFunc = streamCipher.Decrypt
	default:
		text := string(msg.EncryptedData)
		fyne.Do(func() {
			addMessage(list, scroll, Message{
				Username: msg.Username,
				Text:     "[Unknown encryption mode] " + text,
				Image:    nil,
				IsSent:   false,
			})
		})
		return
	}

	decrypted, err := decryptFunc(msg.EncryptedData)
	if err != nil {
		logger.Error("decryption error", "error", err)
		fyne.Do(func() {
			addMessage(list, scroll, Message{
				Username: msg.Username,
				Text:     "[Decryption error]",
				Image:    nil,
				IsSent:   false,
			})
		})
		return
	}

	var text string
	var displayObj fyne.CanvasObject

	if msg.HasImage {
		parts := strings.SplitN(string(decrypted), "\n---CRYPTO_MESSENGER_PPM---\n", 2)
		var ppmBytes []byte
		if len(parts) == 2 {
			text = strings.TrimSpace(parts[0])
			ppmBytes = []byte(parts[1])
		} else {
			text = ""
			ppmBytes = decrypted
		}

		imgDecoded, err := ppm.Decode(strings.NewReader(string(ppmBytes)))
		if err != nil {
			logger.Error("failed to decode received PPM", "error", err)
			text += "\n[Image display error]"
		} else {
			imgObj := canvas.NewImageFromImage(imgDecoded)
			imgObj.FillMode = canvas.ImageFillContain
			imgObj.SetMinSize(fyne.NewSize(500, 400))
			displayObj = imgObj
		}
	} else {
		text = string(decrypted)
	}

	fyne.Do(func() {
		addMessage(list, scroll, Message{
			Username: msg.Username,
			Text:     text,
			Image:    displayObj,
			IsSent:   false,
		})
	})
}

func addMessage(list *fyne.Container, scroll *container.Scroll, msg Message) {
	bubble := container.NewVBox()

	if msg.IsSent {
		bubble.Add(widget.NewLabelWithStyle("You ("+msg.Username+")", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}))
	} else {
		bubble.Add(widget.NewLabelWithStyle("Peer ("+msg.Username+")", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	}

	if msg.Text != "" {
		bubble.Add(widget.NewLabel(msg.Text))
	}

	if msg.Image != nil {
		bubble.Add(msg.Image)
	}

	bubble.Add(widget.NewSeparator())

	if msg.IsSent {
		list.Add(container.NewHBox(container.NewMax(), container.NewPadded(bubble)))
	} else {
		list.Add(container.NewHBox(container.NewPadded(bubble), container.NewMax()))
	}

	list.Refresh()
	scroll.ScrollToBottom()
}
