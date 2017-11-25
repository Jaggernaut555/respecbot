package bot

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
	dbName = "respecdb"
	dbUser = "respecbot"
)

// Global vars
var (
	discordToken   string
	dbPassword     string
	Channels       map[string]bool
	Servers        map[string]bool
	DiscordSession *discordgo.Session
	logger         *log.Logger
)

func initBot() {
	flag.StringVar(&discordToken, "t", "", "Discord Authentication token")
	flag.StringVar(&dbPassword, "p", "", "Password for database user")
	purge := flag.Bool("purge", false, "Use this flag to purge the database. Must be used with -p")
	flag.Parse()

	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)

	InitDB()
	InitRatings()
	InitCmds()
	InitRules()
	InitBets()
	initChannels()

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

func initChannels() {
	Channels = map[string]bool{}
	Servers = map[string]bool{}

	dbLoadActiveChannels(&Channels, &Servers)
}

func LaunchBot() {
	initBot()
	Log("TIME TO RESPEC...")

	if discordToken == "" {
		Log("You must provide a Discord authentication token (-t)")
		return
	}

	var err error
	DiscordSession, err = discordgo.New("Bot " + discordToken)
	if err != nil {
		Log("error creating Discord session,", err.Error())
		return
	}

	// add a handler for when messages are posted
	DiscordSession.AddHandler(messageCreate)
	DiscordSession.AddHandler(reactionAdd)
	DiscordSession.AddHandler(reactionRemove)

	err = DiscordSession.Open()
	if err != nil {
		Log("error opening connection,", err.Error())
		return
	}

	Log("Bot is now running. Press CTRL-C to exit.")
	announceReturn()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	defer DiscordSession.Close()
}

func announceReturn() {
	for k, v := range Channels {
		if v {
			channel, err := DiscordSession.Channel(k)
			if err != nil {
				panic(err)
			}
			if active, ok := Servers[channel.GuildID]; active && ok {
				reply := fmt.Sprintf("I'm back, bitches, and I'm running %v", Version)
				SendReply(k, reply)
				initLosers(channel.GuildID)
				initTopUsers(channel.GuildID)
			}
		}
	}
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	// Do not talk to self
	if message.Author.ID == session.State.User.ID || isBot(message.Author) {
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
	} else if channel != nil && Servers[channel.GuildID] == true && Channels[channel.ID] == true {
		RespecMessage(message.Message)
	}
}

func isBot(user *discordgo.User) bool {
	return user.Bot
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

func Log(data ...string) {
	logger.Print(data)
}
