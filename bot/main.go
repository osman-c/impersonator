package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var session *discordgo.Session

var fillers []string = []string{"the", "this", "is", "are", "and", "to", "in", "at", "from",
	"a", "so", "has", "have", "i", "be", "it", "he", "she"}

func main() {
	log.Println("Starting up...")
	var err error

	log.Println("Connecting to database...")
	initializeDatabase()
	log.Println("Done")

	token := envLookup("BOT_TOKEN")

	log.Println("Initializing session...")
	session, err = discordgo.New("Bot " + token)

	if err != nil {
		log.Fatal(err)
	}
	log.Println("Done!")

	log.Println("Initializing states...")
	initializeStates()
	log.Println("Done!")

	log.Println("Initializing websockets")
	err = session.Open()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Done!")

	session.Identify.Intents = 3072

	log.Println("Initializing commands...")
	InitializeCommands()
	log.Println("Done!")

	log.Println("Initializing command handlers...")
	session.AddHandler(commandHandler)
	log.Println("Done!")

	log.Println("Initializing message handlers...")
	session.AddHandler(messageHandler)
	log.Println("Done!")

	log.Println("Initializing guild handlers...")
	session.AddHandler(guildCreateHandler)
	log.Println("Done!")

	log.Println("Impersonator is ready!")
	sendInAllChannels("Impersonator is ready!")

	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	defer sendInAllChannels("Impersonator is offline.")
	defer session.Close()

	// wtf is this??!?!?
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
