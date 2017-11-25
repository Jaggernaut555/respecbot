package bot

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Jaggernaut555/respecbot/db"
	"github.com/Jaggernaut555/respecbot/state"

	"github.com/Jaggernaut555/respecbot/bet"
	"github.com/Jaggernaut555/respecbot/logging"
	"github.com/Jaggernaut555/respecbot/rate"
	"github.com/bwmarrin/discordgo"
)

// Constants

// Global vars
var (
	discordToken string
	dbPassword   string
)

func initBot() {
	flag.StringVar(&discordToken, "t", "", "Discord Authentication token")
	flag.StringVar(&dbPassword, "p", "", "Password for database user")
	purge := flag.Bool("purge", false, "Use this flag to purge the database. Must be used with -p")

	flag.Parse()

	db.DBSetup(dbPassword, *purge)
	state.InitChannels()
	rate.InitRatings()
	InitCmds()
	rate.InitRules()
	bet.InitBets()

}

func LaunchBot() {
	initBot()
	logging.Log("TIME TO RESPEC...")

	if discordToken == "" {
		logging.Log("You must provide a Discord authentication token (-t)")
		return
	}

	var err error
	state.Session, err = discordgo.New("Bot " + discordToken)
	if err != nil {
		logging.Log("error creating Discord session,", err.Error())
		return
	}

	// add a handler for when messages are posted
	state.Session.AddHandler(messageCreate)
	state.Session.AddHandler(reactionAdd)
	state.Session.AddHandler(reactionRemove)

	err = state.Session.Open()
	if err != nil {
		logging.Log("error opening connection,", err.Error())
		return
	}

	logging.Log("Bot is now running. Press CTRL-C to exit.")
	announceReturn()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	defer state.Session.Close()
}

func announceReturn() {
	for k, v := range state.Channels {
		if v {
			channel, err := state.Session.Channel(k)
			if err != nil {
				panic(err)
			}
			if active, ok := state.Servers[channel.GuildID]; active && ok {
				reply := fmt.Sprintf("I'm back, bitches, and I'm running %v", Version)
				state.SendReply(k, reply)
				rate.InitLosers(channel.GuildID)
				rate.InitTopUsers(channel.GuildID)
			}
		}
	}
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	// Do not talk to self
	if message.Author.ID == session.State.User.ID || message.Author.Bot {
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
	} else if channel != nil && state.Servers[channel.GuildID] == true && state.Channels[channel.ID] == true {
		rate.RespecMessage(message.Message)
	}
}

func reactionAdd(session *discordgo.Session, reaction *discordgo.MessageReactionAdd) {
	rate.RespecReaction(reaction.MessageReaction, true)
}

func reactionRemove(session *discordgo.Session, reaction *discordgo.MessageReactionRemove) {
	rate.RespecReaction(reaction.MessageReaction, false)
}
