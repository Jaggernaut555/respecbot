package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Constants
const (
	Version = "v4.2.0"
)

// Global vars
var (
	token    string
	Channels map[string]bool
	Servers  map[string]bool
)

func init() {
	fmt.Println("TIME TO RESPEC...")
	flag.StringVar(&token, "t", "", "Discord Authentication token")
	flag.Parse()

	Channels = map[string]bool{}
	Servers = map[string]bool{}
	InitRatings()
	InitCmds()
}

func main() {
	if token == "" {
		log.Println("You must provide a Discord authentication token (-t)")
		return
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Println("error creating Discord session,", err)
		return
	}

	// add a handler for when messages are posted
	dg.AddHandler(messageCreate)
	dg.AddHandler(reactionAdd)
	dg.AddHandler(reactionRemove)

	err = dg.Open()
	if err != nil {
		log.Println("error opening connection,", err)
		return
	}

	defer dg.Close()

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	// Do not talk to self
	if message.Author.ID == session.State.User.ID {
		return
	}

	if strings.HasPrefix(message.Content, CmdChar) {
		HandleCommand(session, message, strings.TrimPrefix(message.Content, CmdChar))
		return
	}

	// rate users on everything else they get
	if channel, _ := session.Channel(message.ChannelID); Servers[channel.GuildID] == true {
		RespecMessage(message)
	}
}

func reactionAdd(session *discordgo.Session, reaction *discordgo.MessageReactionAdd) {
	RespecReactionAdd(session, reaction)
}

func reactionRemove(session *discordgo.Session, reaction *discordgo.MessageReactionRemove) {
	RespecReactionRemove(session, reaction)
}

func isValidChannel(session *discordgo.Session, channelID string) bool {
	return Channels[channelID]
}

func SendReply(session *discordgo.Session, channel string, reply string) {
	session.ChannelMessageSend(channel, reply)
}
