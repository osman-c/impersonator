package main

import (
	"errors"
	"math/rand"

	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mb-14/gomarkov"
	"golang.org/x/exp/slices"
)

func FindChain(str string, prev1 string, prev2 string) (string, string, string, error) {
	sameAsBefore := func(first, last string) bool {
		return first == prev1 && last == prev2
	}

	splitted := strings.Split(str, " ")
	for i := len(splitted) - 1; i > 0; i-- {
		for j := i - 1; j >= 0; j-- {
			last := splitted[i]
			first := splitted[j]

			gen, err := state.chain.Generate([]string{first, last})
			if err == nil && !sameAsBefore(first, last) {
				log.Println(first, last)
				return gen, first, last, nil
			}
		}
	}

	for i := len(splitted) - 1; i > 0; i-- {
		for j := i - 1; j >= 0; j-- {
			last := splitted[i]
			first := splitted[j]

			gen, err := state.chain.Generate([]string{last, first})
			if err == nil && !sameAsBefore(last, first) {
				log.Println(last, first)
				return gen, last, first, nil
			}
		}
	}

	for i := len(splitted) - 1; i >= 0; i-- {
		for j := 0; j < len(fillers); j++ {
			last := splitted[i]
			first := fillers[j]

			gen, err := state.chain.Generate([]string{first, last})
			if err == nil && !sameAsBefore(first, last) {
				log.Println(first, last)
				return gen, first, last, nil
			}

			gen, err = state.chain.Generate([]string{last, first})
			if err == nil && !sameAsBefore(last, first) {
				log.Println(last, first)
				return gen, last, first, nil
			}
		}
	}

	return "", "", "", errors.New("no chain found")
}

func FindChainWithOneWord(str string) (string, error) {
	for j := 0; j < len(fillers); j++ {
		gen, err := state.chain.Generate([]string{str, fillers[j]})
		if err == nil {
			log.Println(str, fillers[j])
			return gen, nil
		}

		gen, err = state.chain.Generate([]string{fillers[j], str})
		if err == nil {
			log.Println(fillers[j], str)
			return gen, nil
		}
	}

	return "", errors.New("no chain found")
}

func CanUse(i *discordgo.InteractionCreate) bool {
	if len(state.roles) == 0 {
		return true
	}

	index := slices.IndexFunc[[]string](state.vip,
		func(s string) bool {
			return s == i.Member.User.ID
		})

	if index != -1 {
		return true
	}

	authorID, ok := os.LookupEnv("AUTHOR_ID")
	if ok {
		if authorID == i.Member.User.ID {
			return true
		}
	}

	return IsUserPermitted(i.Member)
}

func Speak(s *discordgo.Session, m *discordgo.MessageCreate, str string, reply bool) {
	var b strings.Builder
	rand.Seed(time.Now().UnixNano())
	limit := state.maxWord
	first, last := "", ""

	for i := 0; i < limit; i++ {
		c, f, l, err := FindChain(fmt.Sprintf(str, b.String()), first, last)
		first, last = f, l
		if err != nil || c == "^" {
			break
		}

		b.WriteString(c)
		if i < limit-1 {
			b.WriteRune(' ')
		}
	}

	content := b.String()

	if reply {
		_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: content,
			Reference: &discordgo.MessageReference{
				MessageID: m.ID,
				ChannelID: m.ChannelID,
				GuildID:   m.GuildID,
			},
		})
		if err != nil {
			log.Println(err)
			return
		}
		state.lastMessageAt = time.Now()
		state.counter = 0
		return
	}

	_, err := s.ChannelMessageSend(m.ChannelID, content)
	if err != nil {
		log.Println(err)
		return
	}
	state.lastMessageAt = time.Now()
	state.counter = 0
}

func Train(s *discordgo.Session) error {
	log.Println("Gathering messages...")
	messages, err := GetAllMessages(s)
	if err != nil {
		return err
	}
	log.Println("Done!")

	chain := gomarkov.NewChain(2)

	log.Println("Training markov chain, n =", len(messages), "...")
	for _, message := range messages {
		chain.Add(strings.Split(message, " "))
		lm := strings.ToLower(message)
		chain.Add(strings.Split(lm, " "))
	}
	log.Println("Done!")

	state.chain = chain
	return nil
}

func SelectUser(s *discordgo.Session, userID string) error {
	if userID == "" {
		return errors.New("no user provided")
	}

	user, err := s.User(userID)
	if err != nil {
		return err
	}

	state.user = user
	_, err = s.UserUpdate(user.Username+" | Impersonator", user.Avatar)
	if err != nil {
		log.Println(err)
	}
	return nil
}

func InitializeCommands(s *discordgo.Session) {
	CreateCommand(s, &discordgo.ApplicationCommand{
		Name:        "set-user",
		Description: "Set user for the bot to impersonate. Options: userID",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "user-id",
				Description: "ID of the user, obtainable by right clicking and selecting 'Copy User ID'",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionString,
			},
		},
	})
	CreateCommand(s, &discordgo.ApplicationCommand{
		Name:        "train",
		Description: "Train markov chain for selected user.",
	})
	CreateCommand(s, &discordgo.ApplicationCommand{
		Name:        "set-limit",
		Description: "Determines how many messages are needed for bot to speak.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "limit",
				Description: "Limit",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionInteger,
			},
		},
	})
	CreateCommand(s, &discordgo.ApplicationCommand{
		Name:        "set-cooldown",
		Description: "Determines how many seconds the next message can be sent.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "cooldown",
				Description: "Cooldown in seconds",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionInteger,
			},
		},
	})
	CreateCommand(s, &discordgo.ApplicationCommand{
		Name:        "add-role",
		Description: "Add role that can use this application.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "role",
				Description: "Name of the role",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionString,
			},
		},
	})
	CreateCommand(s, &discordgo.ApplicationCommand{
		Name:        "remove-role",
		Description: "Remove role that can use this application.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "role",
				Description: "Name of the role",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionString,
			},
		},
	})
	CreateCommand(s, &discordgo.ApplicationCommand{
		Name:        "add-vip",
		Description: "Add vip that can use this application.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "user-id",
				Description: "user-id",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionString,
			},
		},
	})
	CreateCommand(s, &discordgo.ApplicationCommand{
		Name:        "remove-vip",
		Description: "Remove vip that can use this application.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "user-id",
				Description: "user-id",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionString,
			},
		},
	})
	CreateCommand(s, &discordgo.ApplicationCommand{
		Name:        "log-state",
		Description: "Display state of the bot",
	})
}

func CommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if !CanUse(i) {
		EphemeralAlert(s, i.Interaction, "This command is not for you")
		return
	}

	data := i.Interaction.Data.(discordgo.ApplicationCommandInteractionData)
	switch data.Name {
	case "set-user":
		if len(data.Options) < 1 {
			EphemeralAlert(s, i.Interaction, "This command requires at least 1 option")
			return
		}
		userID := data.Options[0].StringValue()
		err := SelectUser(s, userID)

		if err != nil {
			EphemeralAlert(s, i.Interaction, err.Error())
			return
		}
		EphemeralAlert(s, i.Interaction, "User "+userID+" selected!")
		return
	case "train":
		if state.user == nil {
			EphemeralAlert(s, i.Interaction, "You need to set a user first! Use command /set-user")
			return
		}
		err := Train(s)
		if err != nil {
			EphemeralAlert(s, i.Interaction, err.Error())
			return
		}
		EphemeralAlert(s, i.Interaction, "Markov chain trained!")
		return
	case "set-limit":
		if len(data.Options) < 1 {
			EphemeralAlert(s, i.Interaction, "This command requires at least 1 option")
			return
		}
		state.limit = int(data.Options[0].IntValue())
		EphemeralAlert(s, i.Interaction, fmt.Sprintln("New limit is set to", state.limit))
		return
	case "set-cooldown":
		if len(data.Options) < 1 {
			EphemeralAlert(s, i.Interaction, "This command requires at least 1 option")
			return
		}
		value := data.Options[0].IntValue()
		state.cooldown = time.Second * time.Duration(value)
		state.lastMessageAt = time.Now().Add(-24 * time.Hour)
		EphemeralAlert(s, i.Interaction,
			fmt.Sprintln("New cooldown is set to", state.cooldown.Seconds(), "seconds"))
		return
	case "add-role":
		if len(data.Options) < 1 {
			EphemeralAlert(s, i.Interaction, "This command requires at least 1 option")
			return
		}
		role := data.Options[0].StringValue()
		r, err := SearchRole(s, i.GuildID, role)
		if err != nil {
			EphemeralAlert(s, i.Interaction, err.Error())
			return
		}
		if r == -1 {
			EphemeralAlert(s, i.Interaction, "This role doesn't exist")
			return
		}
		if IsPermitted(role) {
			EphemeralAlert(s, i.Interaction, "This role is already permitted")
			return
		}
		state.roles = append(state.roles, role)
		EphemeralAlert(s, i.Interaction, "This role can now use commands")
		return
	case "remove-role":
		if len(data.Options) < 1 {
			EphemeralAlert(s, i.Interaction, "This command requires at least 1 option")
			return
		}
		role := data.Options[0].StringValue()
		if !IsPermitted(role) {
			EphemeralAlert(s, i.Interaction, "This role is already not permitted")
			return
		}
		state.roles = slices.DeleteFunc[[]string](state.roles, func(s string) bool {
			return s == role
		})
		EphemeralAlert(s, i.Interaction, "This role now cam't use commands")
		return
	case "add-vip":
		if len(data.Options) < 1 {
			EphemeralAlert(s, i.Interaction, "This command requires at least 1 option")
			return
		}
		vip := data.Options[0].StringValue()
		_, err := s.User(vip)
		if err != nil {
			EphemeralAlert(s, i.Interaction, err.Error())
			return
		}

		index := slices.IndexFunc[[]string](state.vip, func(s string) bool {
			return s == vip
		})

		if index != -1 {
			EphemeralAlert(s, i.Interaction, "This user is already permitted")
			return
		}

		state.vip = append(state.vip, vip)
		EphemeralAlert(s, i.Interaction, "This user can now use commands")
		return
	case "remove-vip":
		if len(data.Options) < 1 {
			EphemeralAlert(s, i.Interaction, "This command requires at least 1 option")
			return
		}
		vip := data.Options[0].StringValue()
		index := slices.IndexFunc[[]string](state.vip, func(s string) bool {
			return s == vip
		})
		if index == -1 {
			EphemeralAlert(s, i.Interaction, "This user is already not permitted")
			return
		}
		state.vip = slices.DeleteFunc[[]string](state.vip, func(s string) bool {
			return s == vip
		})
		EphemeralAlert(s, i.Interaction, "This user now cam't use commands")
		return
	case "log-state":
		EphemeralAlert(s, i.Interaction, state.String())
	}
}

func MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if state.user == nil || state.chain == nil {
		return
	}

	state.counter++

	next := state.lastMessageAt.Add(time.Duration(state.cooldown))
	if time.Now().Before(next) {
		log.Println("hello")
		log.Println(state.lastMessageAt)
		return
	}

	mentionIndex := slices.IndexFunc[[]*discordgo.User](m.Mentions,
		func(u *discordgo.User) bool {
			return u.ID == s.State.User.ID
		})

	if mentionIndex != -1 {
		content := RemoveMention(m.Message.Content)
		Speak(s, m, content, true)
		return
	}

	if m.ReferencedMessage != nil {
		replied := m.ReferencedMessage.Author.ID == s.State.User.ID
		if replied {
			Speak(s, m, m.Message.Content, true)
			return
		}
	}

	if state.counter > state.limit {
		Speak(s, m, m.Message.Content, false)
		return
	}
}
