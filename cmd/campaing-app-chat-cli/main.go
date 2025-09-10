package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	authv1 "github.com/Venqis-NolaTech/campaing-app-auth-api-go/proto/generated/services/auth/v1"
	authv1client "github.com/Venqis-NolaTech/campaing-app-auth-api-go/proto/generated/services/auth/v1/client"
	chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
	chatv1client "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1/client"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/utils"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api/auth"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
)

var generalParams api.GeneralParams

var userID = flag.Int("u", 0, "user id")
var remitentID = flag.Int("t", 0, "remitent id")
var roomID = flag.Int("r", 0, "room id")

var room *chatv1.Room
var me *authv1.MeResponse

func main() {
	flag.Parse()

	if userID == nil || *userID == 0 {
		log.Fatal("el usuario es un campo requerido")
	}

	token, err := auth.GenerateSessionToken(auth.SessionData{
		UserID: *userID,
		Type:   "ACCESS",
	})

	session, _ := auth.ValidateSessionToken(token)

	location, err := time.LoadLocation("Local")
	if err != nil {
		log.Fatal("Can't get timezone: ", err)
		return
	}

	generalParams = api.GeneralParams{
		SessionToken: token,
		Lang:         "es",
		Platform:     "cli",
		IANATimezone: location.String(),
		Session:      session,
		ClientId:     uuid.NewString(),
	}

	res, err := authv1client.Me(context.Background(), generalParams, &authv1.MeRequest{})
	if err != nil {
		log.Fatal("Can't get user data: ", err)
	}
	me = res

	if roomID == nil || *roomID == 0 {
		if remitentID == nil || *remitentID == 0 {
			log.Fatal("el campo de remitente es requerido")
		}
		res, err := chatv1client.CreateRoom(context.Background(), generalParams, &chatv1.CreateRoomRequest{
			Type: "p2p",
			Participants: []int32{
				int32(*remitentID),
			},
		})
		if err != nil {
			log.Fatal("Can't create room: ", err)
		}
		room = res.Room
	} else {
		res, err := chatv1client.GetRoom(context.Background(), generalParams, &chatv1.GetRoomRequest{
			Id: strconv.Itoa(*roomID),
		})
		if err != nil {
			log.Fatal("Can't join room: ", err)
		}
		room = res.Room
	}

	log.Println("Conectando al room:", room.Id)

	// Crea el directorio para el log si no existe
	if err := os.MkdirAll("tmp", 0755); err != nil {
		log.Fatal(err)
	}

	// Configura un archivo de log para depuración
	f, err := tea.LogToFile("tmp/chat-debug.log", "debug")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Inicia el programa de Bubble Tea
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

// message es un tipo para los mensajes recibidos del canal de escucha.
// Usar un tipo propio nos permite distinguirlo de otros strings.
type message string

// errMessage se usa para comunicar errores al ciclo de Update.
type errMessage error

// sessionState contiene el estado de nuestra sesión de chat.
type sessionState struct {
	ctx         context.Context
	cancel      context.CancelFunc
	messages    []string
	messageChan chan message
	err         error
}

type model struct {
	state       sessionState
	viewport    viewport.Model
	textarea    textarea.Model
	senderStyle lipgloss.Style
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Escribe tu mensaje y presiona Enter..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false) // Deshabilita el salto de línea con Enter

	vp := viewport.New(80, 30)
	vp.SetContent(fmt.Sprintf("Estás en la sala %s\nPresiona Ctrl+C para salir.\n\n Información del usuario: %s", room.Id, protojson.Format(me)))

	// Estado inicial de la sesión
	ctx, cancel := context.WithCancel(context.Background())
	initialState := sessionState{
		ctx:         ctx,
		cancel:      cancel,
		messages:    []string{},
		messageChan: make(chan message),
	}

	return model{
		state:       initialState,
		viewport:    vp,
		textarea:    ta,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
	}
}

func listenForMessages(ctx context.Context, msgChan chan<- message) {
	req, _ := api.NewRequest(generalParams, &chatv1.StreamMessagesRequest{})
	client := chatv1client.GetChatServiceClient()

	stream, err := client.StreamMessages(ctx, req)
	if err != nil {
		log.Fatal("Can't stream messages: ", err)
	}
	defer stream.Close()

	for stream.Receive() {
		if err := stream.Err(); err != nil {
			log.Println("Error with stream:", err)
			continue
		}
		log.Println("Event received for user:", me.Id, protojson.Format(stream.Msg()))

		msg := stream.Msg().GetMessage()
		if msg != nil && msg.RoomId == room.Id && uint32(msg.GetSenderId()) != me.Id {
			contact := msg.GetContactName()
			content, err := utils.DecryptMessage(msg.Content, room.EncryptionData)
			if err != nil {
				log.Println("Error decripting message:", err)
			}
			msgChan <- message(fmt.Sprintf("%s: %s", contact, content))
		}
	}

	log.Println("Cerrando el listener de mensajes.")
}

func sendMessage(content string) {
	message, err := utils.EncryptMessage(content, room.EncryptionData)
	if err != nil {
		fmt.Println("Error enviando mensaje: ", err)
		return
	}
	_, err = chatv1client.SendMessage(context.Background(), generalParams, &chatv1.SendMessageRequest{
		Type:         "user",
		RoomId:       room.Id,
		Content:      message,
		ContactName:  &me.Name,
		ContactPhone: &me.Phone,
	})
	if err != nil {
		fmt.Println("Error enviando mensaje: ", err)
		return
	}
	log.Printf("Mensaje a enviar: '%s'", content)
}

func (m *model) waitForMessage() tea.Cmd {
	return func() tea.Msg {
		return <-m.state.messageChan
	}
}

func (m model) Init() tea.Cmd {
	go listenForMessages(m.state.ctx, m.state.messageChan)
	return m.waitForMessage()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		cmd   tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.state.cancel()
			return m, tea.Quit
		case tea.KeyEnter:
			userInput := m.textarea.Value()
			if userInput != "" {
				sendMessage(userInput)
				m.state.messages = append(m.state.messages, m.senderStyle.Render("Tú: ")+userInput)
				m.viewport.SetContent(strings.Join(m.state.messages, "\n"))
				m.textarea.Reset()
				m.viewport.GotoBottom()
			}
		}

	case message:
		m.state.messages = append(m.state.messages, string(msg))
		m.viewport.SetContent(strings.Join(m.state.messages, "\n"))
		m.viewport.GotoBottom()
		cmd = m.waitForMessage()

	case errMessage:
		m.state.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd, cmd)
}

func (m model) View() string {
	if m.state.err != nil {
		return fmt.Sprintf("Error: %v", m.state.err)
	}

	return fmt.Sprintf(`%s

%s`,
		m.viewport.View(),
		m.textarea.View(),
	) + "\n"
}
