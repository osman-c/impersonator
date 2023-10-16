package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slices"
)

func EnvLookup(str string) string {
	res, ok := os.LookupEnv(str)
	if !ok {
		log.Fatalf("No %v in env\n", str)
	}

	return res
}

func IsPermitted(role string) bool {
	for _, p := range state.roles {
		if p == role {
			return true
		}
	}
	return false
}

func IsUserPermitted(member *discordgo.Member) bool {
	for _, role := range member.Roles {
		if IsPermitted(role) {
			return true
		}
	}
	return false
}

func CreateCommand(s *discordgo.Session, cmd *discordgo.ApplicationCommand) {
	applicationID := EnvLookup("APPLICATION_ID")

	_, err := s.ApplicationCommandCreate(applicationID, "", cmd)

	if err != nil {
		log.Println("Command couldn't be initialized: ", cmd.Name)
		log.Println(err)
		return
	}
}

func SearchRole(s *discordgo.Session, guildID string, role string) (int, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return 0, err
	}

	return slices.IndexFunc[[]*discordgo.Role](roles, func(r *discordgo.Role) bool {
		return r.Name == role
	}), nil
}

func EphemeralAlert(s *discordgo.Session, interaction *discordgo.Interaction, content string) {
	s.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   64, // ephemeral btw
		},
	})
}

func removeSpecialCharacters(input *string) {
	// Define a regular expression to match non-alphanumeric characters
	regex := regexp.MustCompile("[^a-zA-Z0-9]+")

	// Replace all non-alphanumeric characters with an empty string
	cleaned := regex.ReplaceAllString(*input, "")
	*input = cleaned
}

func RemoveMention(str string) string {
	// Create a regular expression pattern to match the substring
	// The `regexp.QuoteMeta` function is used to escape any special characters in the substring
	splitted := strings.Split(str, " ")
	splitted = slices.DeleteFunc[[]string](splitted,
		func(s string) bool {
			return s[0] == '<'
		})

	return strings.Join(splitted, " ")
}

func Purify(splitted *([]string)) {
	for _, word := range *splitted {
		removeSpecialCharacters(&word)
	}
}

func GetAllChannelMessages(
	s *discordgo.Session,
	id string, contents *[]string,
	filter func(m *discordgo.Message) bool) (int, int) {

	before := ""
	total, max := 0, 0

	for {
		messages, err := s.ChannelMessages(id, 100, before, "", "")

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
		log.Println("hello")
	}

}

func GetAllChannels(s *discordgo.Session) ([]*discordgo.Channel, error) {
	res := []*discordgo.Channel{}

	guilds, err := s.UserGuilds(100, "", "")
	if err != nil {
		return nil, err
	}

	for _, guild := range guilds {
		channels, err := s.GuildChannels(guild.ID)
		if err != nil {
			log.Println(err)
		}

		res = append(res, channels...)
	}

	return res, nil
}

func SendInAllChannels(s *discordgo.Session, content string) {
	channels, _ := GetAllChannels(s)

	for _, channel := range channels {
		s.ChannelMessageSend(channel.ID, content)
	}
}

func ContainsRole(userRoles []string, roleID string) bool {
	for _, role := range userRoles {
		if role == roleID {
			return true
		}
	}
	return false
}

func GetAllMessages(s *discordgo.Session) ([]string, error) {
	if state.user == nil {
		return nil, errors.New("no user set")
		// s.ChannelMessageSend()
	}

	messages := []string{}

	channels, err := GetAllChannels(s)
	if err != nil {
		return nil, err
	}

	total, max := 0, 0

	for _, channel := range channels {
		t, m := GetAllChannelMessages(s, channel.ID, &messages, func(m *discordgo.Message) bool {
			return state.user.ID == m.Author.ID
		})

		if m > max {
			max = m
		}
		total += t
	}

	if len(messages) > 0 {
		state.maxWord = max
	}
	return messages[:], err
}
