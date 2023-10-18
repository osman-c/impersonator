package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mb-14/gomarkov"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/exp/slices"
)

type User interface {
	User()
}

type State struct {
	GuildID       string
	User          string
	Chain         []byte
	MaxWord       int
	Counter       int
	Limit         int
	Cooldown      time.Duration
	LastMessageAt time.Time
	Roles         []string
	Vip           []string
}

func makeDefaultState(guildID string) *State {
	return &State{
		GuildID:       guildID,
		User:          "",
		Chain:         []byte{},
		MaxWord:       0,
		Counter:       0,
		Limit:         20,
		Cooldown:      time.Second * 10,
		LastMessageAt: time.Now().Add(-24 * time.Hour),
		Roles:         []string{},
		Vip:           []string{},
	}
}

func (s *State) Update(k string, v any) error {
	filter := bson.D{{Key: "guildid", Value: s.GuildID}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: k, Value: v}}}}

	_, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	err = coll.FindOne(ctx, filter).Decode(s)
	if err != nil {
		return err
	}

	return nil
}

func (s *State) IsPermitted(role string) bool {
	for _, p := range s.Roles {
		if p == role {
			return true
		}
	}
	return false
}

func (s *State) IsUserPermitted(member *discordgo.Member) bool {
	for _, role := range member.Roles {
		if s.IsPermitted(role) {
			return true
		}
	}
	return false
}

func (s *State) CanUse(i *discordgo.InteractionCreate) bool {
	if len(s.Roles) == 0 {
		return true
	}

	index := slices.IndexFunc[[]string](s.Vip,
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

	return s.IsUserPermitted(i.Member)
}

func (s *State) GetAllMessages() ([]string, error) {
	if s.User == "" {
		return nil, errors.New("no user set")
		// s.ChannelMessageSend()
	}

	messages := []string{}

	channels, err := session.GuildChannels(s.GuildID)
	if err != nil {
		return nil, err
	}

	total, max := 0, 0

	for _, channel := range channels {
		t, m := getAllChannelMessages(channel.ID, &messages,
			func(m *discordgo.Message) bool {
				return s.User == m.Author.ID
			})

		if m > max {
			max = m
		}
		total += t
	}

	if len(messages) > 0 {
		s.Update("maxword", max)
	}
	return messages[:], err
}

func (s *State) Train() error {
	log.Println("Gathering messages...")
	messages, err := s.GetAllMessages()
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

	json, err := chain.MarshalJSON()
	if err != nil {
		return err
	}

	s.Update("chain", json)
	return nil
}

func (s *State) FindChain(chain *gomarkov.Chain, str string, prev1 string, prev2 string) (string, string, string, error) {
	sameAsBefore := func(first, last string) bool {
		return first == prev1 && last == prev2
	}

	splitted := strings.Split(str, " ")
	for i := len(splitted) - 1; i > 0; i-- {
		for j := i - 1; j >= 0; j-- {
			last := splitted[i]
			first := splitted[j]

			gen, err := chain.Generate([]string{first, last})
			if err == nil && !sameAsBefore(first, last) {
				return gen, first, last, nil
			}
		}
	}

	for i := len(splitted) - 1; i > 0; i-- {
		for j := i - 1; j >= 0; j-- {
			last := splitted[i]
			first := splitted[j]

			gen, err := chain.Generate([]string{last, first})
			if err == nil && !sameAsBefore(last, first) {
				return gen, last, first, nil
			}
		}
	}

	for i := len(splitted) - 1; i >= 0; i-- {
		for j := 0; j < len(fillers); j++ {
			last := splitted[i]
			first := fillers[j]

			gen, err := chain.Generate([]string{first, last})
			if err == nil && !sameAsBefore(first, last) {
				return gen, first, last, nil
			}

			gen, err = chain.Generate([]string{last, first})
			if err == nil && !sameAsBefore(last, first) {
				return gen, last, first, nil
			}
		}
	}

	return "", "", "", errors.New("no chain found")
}

func (s *State) Speak(m *discordgo.MessageCreate, reply bool, limit int, callback func()) error {
	var chain gomarkov.Chain
	err := json.Unmarshal(s.Chain, &chain)
	if err != nil {
		return err
	}

	var b strings.Builder

	rand.Seed(time.Now().UnixNano())
	first, last := "", ""

	str := removeMention(m.Content)
	str = removeNonAlphaSpace(str)

	for i := 0; i < limit; i++ {
		c, f, l, err := s.FindChain(&chain, fmt.Sprintf(str, b.String()), first, last)
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
	if content == "" {
		return errors.New("chain not found")
	}

	if reply {
		_, err := session.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: content,
			Reference: &discordgo.MessageReference{
				MessageID: m.ID,
				ChannelID: m.ChannelID,
				GuildID:   m.GuildID,
			},
		})
		if err != nil {
			return err
		}
		callback()
		return nil
	}

	_, err = session.ChannelMessageSend(m.ChannelID, content)
	if err != nil {
		return err
	}
	s.LastMessageAt = time.Now()
	s.Counter = 0
	return nil
}
