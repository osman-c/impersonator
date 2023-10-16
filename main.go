package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/mb-14/gomarkov"
)

type WordCount struct {
	min, max, avg int
}

type State struct {
	user          *discordgo.User
	chain         *gomarkov.Chain
	maxWord       int
	counter       int
	limit         int
	cooldown      time.Duration
	lastMessageAt time.Time
	roles         []string
	vip           []string
}

func (s *State) String() string {
	var sb strings.Builder
	if state.user != nil {
		sb.WriteString(fmt.Sprintln("Impersonated user:", state.user.Username))
	} else {
		sb.WriteString(fmt.Sprintln("Inpoersonated user: None"))
	}
	if state.chain != nil {
		sb.WriteString(fmt.Sprintln("Chain: Present"))
	} else {
		sb.WriteString(fmt.Sprintln("Chain: None"))
	}
	sb.WriteString(fmt.Sprintln("Limit:", state.limit))
	sb.WriteString(fmt.Sprintln("Cooldown:", state.cooldown))
	sb.WriteString(fmt.Sprintln("Permitted Roles: ", state.roles))

	return sb.String()
}

var state State = State{
	user:          nil,
	chain:         nil,
	maxWord:       20,
	counter:       0,
	limit:         20,
	cooldown:      time.Second * 10,
	lastMessageAt: time.Now().Add(-24 * time.Hour),
	roles:         []string{},
	vip:           []string{},
}

var fillers []string = []string{"the", "this", "is", "are", "and", "to", "in", "at", "from",
	"a", "so", "has", "have", "i", "be", "it", "he", "she"}

func main() {
	log.Println("Starting up...")
	if err := godotenv.Load(); err != nil {
		log.Fatal("No .env file found")
	}

	token := EnvLookup("BOT_TOKEN")

	log.Println("Initializing session...")
	session, err := discordgo.New("Bot " + token)

	if err != nil {
		log.Fatal(err)
	}
	log.Println("Done!")

	log.Println("Initializing websockets")
	err = session.Open()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Done!")

	session.Identify.Intents = 3072

	log.Println("Initializing commands...")
	InitializeCommands(session)
	log.Println("Done!")

	log.Println("Initializing command handlers...")
	session.AddHandler(CommandHandler)
	log.Println("Done!")

	log.Println("Initializing message handlers...")
	session.AddHandler(MessageHandler)
	log.Println("Done!")

	SendInAllChannels(session, "Impersonator is ready")
	log.Println("Impersonator is ready")

	defer session.Close()

	// wtf is this??!?!?
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
