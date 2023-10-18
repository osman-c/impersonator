package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slices"
)

func InitializeCommands() {
	createCommand(&discordgo.ApplicationCommand{
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
	createCommand(&discordgo.ApplicationCommand{
		Name:        "train",
		Description: "Train markov chain for selected user.",
	})
	createCommand(&discordgo.ApplicationCommand{
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
	createCommand(&discordgo.ApplicationCommand{
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
	createCommand(&discordgo.ApplicationCommand{
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
	createCommand(&discordgo.ApplicationCommand{
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
	createCommand(&discordgo.ApplicationCommand{
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
	createCommand(&discordgo.ApplicationCommand{
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
	createCommand(&discordgo.ApplicationCommand{
		Name:        "log-state",
		Description: "Display state of the bot",
	})
}

func commandHandler(session *discordgo.Session, i *discordgo.InteractionCreate) {
	state, err := getState(i.GuildID)
	if err != nil {
		log.Println(err)
		return
	}

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if !state.CanUse(i) {
		ephemeralAlert(i.Interaction, "This command is not for you")
		return
	}

	data := i.Interaction.Data.(discordgo.ApplicationCommandInteractionData)
	switch data.Name {
	case "set-user":
		if len(data.Options) < 1 {
			ephemeralAlert(i.Interaction, "This command requires at least 1 option")
			return
		}
		userID := data.Options[0].StringValue()
		err := state.Update("user", userID)

		if err != nil {
			ephemeralAlert(i.Interaction, err.Error())
			return
		}
		ephemeralAlert(i.Interaction, "User "+userID+" selected!")
		return
	case "train":
		if state.User == "" {
			ephemeralAlert(i.Interaction, "You need to set a user first! Use command /set-user")
			return
		}
		err := state.Train()
		if err != nil {
			ephemeralAlert(i.Interaction, err.Error())
			return
		}
		ephemeralAlert(i.Interaction, "Markov chain trained!")
		return
	case "set-limit":
		if len(data.Options) < 1 {
			ephemeralAlert(i.Interaction, "This command requires at least 1 option")
			return
		}
		state.Limit = int(data.Options[0].IntValue())
		ephemeralAlert(i.Interaction, fmt.Sprintln("New limit is set to", state.Limit))
		return
	case "set-cooldown":
		if len(data.Options) < 1 {
			ephemeralAlert(i.Interaction, "This command requires at least 1 option")
			return
		}
		value := data.Options[0].IntValue()
		state.Cooldown = time.Second * time.Duration(value)
		state.LastMessageAt = time.Now().Add(-24 * time.Hour)
		ephemeralAlert(i.Interaction,
			fmt.Sprintln("New cooldown is set to", state.Cooldown.Seconds(), "seconds"))
		return
	case "add-role":
		if len(data.Options) < 1 {
			ephemeralAlert(i.Interaction, "This command requires at least 1 option")
			return
		}
		role := data.Options[0].StringValue()
		r, err := searchRole(i.GuildID, role)
		if err != nil {
			ephemeralAlert(i.Interaction, err.Error())
			return
		}
		if r == -1 {
			ephemeralAlert(i.Interaction, "This role doesn't exist")
			return
		}
		if !state.IsPermitted(role) {
			ephemeralAlert(i.Interaction, "This role is already permitted")
			return
		}
		state.Roles = append(state.Roles, role)
		ephemeralAlert(i.Interaction, "This role can now use commands")
		return
	case "remove-role":
		if len(data.Options) < 1 {
			ephemeralAlert(i.Interaction, "This command requires at least 1 option")
			return
		}
		role := data.Options[0].StringValue()
		if !state.IsPermitted(role) {
			ephemeralAlert(i.Interaction, "This role is already not permitted")
			return
		}
		state.Roles = slices.DeleteFunc[[]string](state.Roles, func(s string) bool {
			return s == role
		})
		ephemeralAlert(i.Interaction, "This role now cam't use commands")
		return
	case "add-vip":
		if len(data.Options) < 1 {
			ephemeralAlert(i.Interaction, "This command requires at least 1 option")
			return
		}
		vip := data.Options[0].StringValue()
		_, err := session.User(vip)
		if err != nil {
			ephemeralAlert(i.Interaction, err.Error())
			return
		}

		index := slices.IndexFunc[[]string](state.Vip, func(s string) bool {
			return s == vip
		})

		if index != -1 {
			ephemeralAlert(i.Interaction, "This user is already permitted")
			return
		}

		state.Vip = append(state.Vip, vip)
		ephemeralAlert(i.Interaction, "This user can now use commands")
		return
	case "remove-vip":
		if len(data.Options) < 1 {
			ephemeralAlert(i.Interaction, "This command requires at least 1 option")
			return
		}
		vip := data.Options[0].StringValue()
		index := slices.IndexFunc[[]string](state.Vip, func(s string) bool {
			return s == vip
		})
		if index == -1 {
			ephemeralAlert(i.Interaction, "This user is already not permitted")
			return
		}
		state.Vip = slices.DeleteFunc[[]string](state.Vip, func(s string) bool {
			return s == vip
		})
		ephemeralAlert(i.Interaction, "This user now cam't use commands")
		return
	}
}

func messageHandler(session *discordgo.Session, m *discordgo.MessageCreate) {
	state, err := getState(m.GuildID)
	if err != nil {
		log.Println(err)
		return
	}

	if m.Author.ID == session.State.User.ID {
		return
	}

	if state.User == "" || len(state.Chain) == 0 {
		return
	}

	err = state.Update("Counter", state.Counter+1)
	if err != nil {
		log.Println(err)
	}

	next := state.LastMessageAt.Add(time.Duration(state.Cooldown))
	if time.Now().Before(next) {
		return
	}

	mentionIndex := slices.IndexFunc[[]*discordgo.User](m.Mentions,
		func(u *discordgo.User) bool {
			return u.ID == session.State.User.ID
		})

	callback := func() {
		err = state.Update("Counter", 0)
		if err != nil {
			log.Println(err)
		}

		err = state.Update("LastMessageAt", time.Now())
		if err != nil {
			log.Println(err)
		}
	}

	if mentionIndex != -1 {
		err := state.Speak(m, true, state.Limit, callback)
		if err != nil {
			log.Println(err)
		}
		return
	}

	if m.ReferencedMessage != nil {
		replied := m.ReferencedMessage.Author.ID == session.State.User.ID
		if replied {
			err := state.Speak(m, true, state.Limit, callback)
			if err != nil {
				log.Println(err)
			}
			return
		}
	}

	if state.Counter > state.Limit {
		err := state.Speak(m, false, state.Limit, callback)
		if err != nil {
			log.Println(err)
		}
		return
	}
}

func guildCreateHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
	state := makeDefaultState(g.Guild.ID)
	_, err := coll.InsertOne(ctx, *state)
	if err != nil {
		log.Println(err)
	}

	for _, channel := range g.Channels {
		s.ChannelMessageSend(channel.ID, "Impersonator just joined this server! Use '/set-user' to set a user to impersonate and use '/train' to train a markov chain.")
	}
}
