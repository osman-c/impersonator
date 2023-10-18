package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slices"
)

func envLookup(str string) string {
	res, ok := os.LookupEnv(str)
	if !ok {
		log.Fatalf("No %v in env\n", str)
	}

	return res
}

func createCommand(cmd *discordgo.ApplicationCommand) {
	applicationID := envLookup("APPLICATION_ID")

	_, err := session.ApplicationCommandCreate(applicationID, "", cmd)

	if err != nil {
		log.Println("Command couldn't be initialized: ", cmd.Name)
		log.Println(err)
		return
	}
}

func searchRole(guildID string, role string) (int, error) {
	roles, err := session.GuildRoles(guildID)
	if err != nil {
		return 0, err
	}

	return slices.IndexFunc[[]*discordgo.Role](roles, func(r *discordgo.Role) bool {
		return r.Name == role
	}), nil
}

func ephemeralAlert(interaction *discordgo.Interaction, content string) {
	session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   64, // ephemeral btw
		},
	})
}

func removeMention(str string) string {
	splitted := strings.Split(str, " ")
	splitted = slices.DeleteFunc[[]string](splitted,
		func(s string) bool {
			return s[0] == '<'
		})

	return strings.Join(splitted, " ")
}

func removeNonAlphaSpace(input string) string {
	regex := regexp.MustCompile("[^A-Za-z ]")

	cleaned := regex.ReplaceAllString(input, "")

	return cleaned
}

func getAllChannelMessages(
	id string, contents *[]string,
	filter func(m *discordgo.Message) bool) (int, int) {

	before := ""
	total, max := 0, 0

	for {
		messages, err := session.ChannelMessages(id, 100, before, "", "")

		if err != nil {
			fmt.Println(err)
			return total, max
		}

		if len(messages) == 0 {
			return total, max
		}

		for _, message := range messages {
			correctType := message.Type == discordgo.MessageTypeDefault || message.Type == discordgo.MessageTypeReply
			splitted := strings.Split(message.Content, " ")
			correctLength := len(splitted) > 2
			if correctLength && correctType && filter(message) {
				*contents = append(*contents, message.Content)
			}

			if max < len(splitted) {
				max = len(splitted)
			}
			total += len(splitted)
		}

		before = messages[len(messages)-1].ID
	}
}

func sendInAllChannels(content string) error {
	guilds, err := session.UserGuilds(100, "", "")
	if err != nil {
		return err
	}

	for _, guild := range guilds {
		channels, _ := session.GuildChannels(guild.ID)
		for _, channel := range channels {
			session.ChannelMessageSend(channel.ID, content)
		}
	}

	return nil
}
