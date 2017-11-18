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
	//_ "github.com/go-sql-driver/mysql"
)

// Constants
const (
	Version = "v4.2.0"
	dbName  = "respecdb"
	dbUser  = "respecbot"
)

// Global vars
var (
	discordToken   string
	dbPassword     string
	Channels       map[string]bool
	Servers        map[string]bool
	DiscordSession *discordgo.Session
)

func init() {
	flag.StringVar(&discordToken, "t", "", "Discord Authentication token")
	flag.StringVar(&dbPassword, "p", "", "Password for database user")
	purge := flag.Bool("purge", false, "Use this flag to purge the database. Must be used with -p")
	flag.Parse()

	log.SetOutput(os.Stdout)

	Channels = map[string]bool{}
	Servers = map[string]bool{}
	InitDB()
	InitRatings()
	InitCmds()
	InitRules()
	InitBets()

	if *purge {
		if dbPassword != "" {
			if err := purgeDB(); err != nil {
				panic(err)
			}
			os.Exit(1)
		} else {
			fmt.Print("Please provide a valid database password with -p")
			os.Exit(1)
		}
	}
}

func main() {
	log.Println("TIME TO RESPEC...")

	if discordToken == "" {
		log.Println("You must provide a Discord authentication token (-t)")
		return
	}

	var err error
	DiscordSession, err = discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Println("error creating Discord session,", err)
		return
	}

	// add a handler for when messages are posted
	DiscordSession.AddHandler(messageCreate)
	DiscordSession.AddHandler(reactionAdd)
	DiscordSession.AddHandler(reactionRemove)

	err = DiscordSession.Open()
	if err != nil {
		log.Println("error opening connection,", err)
		return
	}

	log.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	defer DiscordSession.Close()
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	// Do not talk to self
	if message.Author.ID == session.State.User.ID {
		return
	}

	if strings.HasPrefix(message.Content, CmdChar) {
		HandleCommand(message, strings.TrimPrefix(message.Content, CmdChar))
		return
	}

	// rate users on everything else they get
	channel, err := session.Channel(message.ChannelID)
	if err != nil {
		return
	} else if channel != nil && Servers[channel.GuildID] == true {
		RespecMessage(message.Message)
	}
}

func reactionAdd(session *discordgo.Session, reaction *discordgo.MessageReactionAdd) {
	RespecReaction(reaction.MessageReaction, true)
}

func reactionRemove(session *discordgo.Session, reaction *discordgo.MessageReactionRemove) {
	RespecReaction(reaction.MessageReaction, false)
}

func isValidChannel(channelID string) bool {
	return Channels[channelID]
}

func SendReply(channelID string, reply string) {
	DiscordSession.ChannelMessageSend(channelID, reply)
}

func SendEmbed(channelID string, embed *discordgo.MessageEmbed) (msg *discordgo.Message) {
	msg, _ = DiscordSession.ChannelMessageSendEmbed(channelID, embed)
	return
}
