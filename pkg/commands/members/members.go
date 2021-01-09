package members

import (
	"fmt"
	"log"
	"time"

	"github.com/itfactory-tm/thomas-bot/pkg/embed"

	"github.com/bwmarrin/discordgo"
	"github.com/itfactory-tm/thomas-bot/pkg/command"
)

// TODO: replace me
const itfDiscord = "687565213943332875"
const itfWelcome = "687588438886842373"
const guestRole = "687568536356257890"

// MemberCommands contains the tm!role command and welcome messages
type MemberCommands struct{}

// NewMemberCommand gives a new MemberCommands
func NewMemberCommand() *MemberCommands {
	return &MemberCommands{}
}

// Register registers the handlers
func (m *MemberCommands) Register(registry command.Registry, server command.Server) {
	registry.RegisterMessageCreateHandler("role", m.sayRole)
	registry.RegisterGuildMemberAddHandler(m.onGuildMemberAdd)
	registry.RegisterMessageReactionAddHandler(m.handleRolePermissionReaction)
	registry.RegisterMessageReactionAddHandler(m.handleRoleReaction)
}

func (m *MemberCommands) onGuildMemberAdd(s *discordgo.Session, g *discordgo.GuildMemberAdd) {
	if g.GuildID != itfDiscord {
		return
	}

	err := s.GuildMemberRoleAdd(g.GuildID, g.Member.User.ID, guestRole) // gast role
	if err != nil {
		log.Printf("Cannot set role for user %s: %q\n", g.Member.User.ID, err)
	}

	welcome, _ := s.ChannelMessageSend(itfWelcome, fmt.Sprintf("Welcome <@%s> to the **IT Factory Official** Discord server. We will send you a DM in a moment to get you set up!", g.User.ID))

	c, err := s.UserChannelCreate(g.Member.User.ID)
	if err != nil {
		log.Printf("Cannot DM user %s\n", g.Member.User.ID)
		return
	}

	s.ChannelMessageSend(c.ID, fmt.Sprintf("Hello %s", g.User.Username))
	time.Sleep(time.Second)
	s.ChannelMessageSend(c.ID, "Welcome to the ITFactory Discord!")
	time.Sleep(time.Second)
	s.ChannelMessageSend(c.ID, "My name is Thomas Bot, I am a bot who can help you!")
	time.Sleep(time.Second)
	s.ChannelMessageSend(c.ID, "New to Discord? No problem we got a manual for you: https://itf.to/discord-help")
	embed := embed.NewEmbed()
	embed.SetImage("https://static.eyskens.me/thomas-bot/opendeurdag-1.png")
	embed.SetURL("https://itf.to/discord-help")
	s.ChannelMessageSendEmbed(c.ID, embed.MessageEmbed)

	time.Sleep(time.Second)
	s.ChannelMessageSend(c.ID, "If you need help just type tm!help")
	time.Sleep(time.Second)
	s.ChannelMessageSend(c.ID, "Warning, i am only able to reply to messages starting with `tm!`, not to normal questions.")
	time.Sleep(5 * time.Second)
	s.ChannelMessageSend(c.ID, "Please set your name for our Discord server to your actual name, this will help us to identify you and let you in! Thank you!")
	time.Sleep(3 * time.Second)

	m.SendRoleDM(s, g.Member.User.ID)

	time.Sleep(time.Minute)
	err = s.MessageReactionAdd(itfWelcome, welcome.ID, "797537339613249567")
	if err != nil {
		log.Println(err)
	}
}

// Info return the commands in this package
func (m *MemberCommands) Info() []command.Command {
	return []command.Command{
		command.Command{
			Name:        "role",
			Category:    command.CategoryAlgemeen,
			Description: "Modify your ITFactory Discord role",
			Hidden:      false,
		},
	}
}
