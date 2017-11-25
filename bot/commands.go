package bot

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Jaggernaut555/respecbot/cards"
	"github.com/Jaggernaut555/respecbot/db"
	"github.com/Jaggernaut555/respecbot/games"
	"github.com/Jaggernaut555/respecbot/rate"
	"github.com/Jaggernaut555/respecbot/state"
	"github.com/bwmarrin/discordgo"
)

// Constants
const (
	CmdChar = "%"
)

// CmdFuncType Command function type
type CmdFuncType func(*discordgo.MessageCreate, []string)

// CmdFuncHelpType The type stored in the CmdFuncs map to map a function and helper text to a command
type CmdFuncHelpType struct {
	function           CmdFuncType
	help               string
	allowedChannelOnly bool
}

// CmdFuncsType The type of the CmdFuncs map
type CmdFuncsType map[string]CmdFuncHelpType

// CmdFuncs Commands to functions map
var CmdFuncs CmdFuncsType

// Initializes the cmds map
func init() {
	CmdFuncs = CmdFuncsType{
		"help":     CmdFuncHelpType{cmdHelp, "Prints this list", false},
		"lookatme": CmdFuncHelpType{cmdHere, "Fuck off, user", false},
		"fuckoff":  CmdFuncHelpType{cmdNotHere, "Fuck off, bot", true},
		"version":  CmdFuncHelpType{cmdVersion, "Outputs the current bot version", true},
		"stats":    CmdFuncHelpType{cmdStats, "Displays stats about this bot", true},
		"bet":      CmdFuncHelpType{cmdBet, "WHO GONNA WIN? `bet help`", true},
		"card":     CmdFuncHelpType{cmdCard, "IS A CARD", true},
	}
}

func HandleCommand(message *discordgo.MessageCreate, cmd string) {
	args := strings.Split(cmd, " ")
	if len(args) == 0 {
		return
	}
	CmdFuncHelpPair, ok := CmdFuncs[args[0]]

	if ok {
		if !CmdFuncHelpPair.allowedChannelOnly || state.IsValidChannel(message.ChannelID) {
			CmdFuncHelpPair.function(message, args)
		}
	} else if state.IsValidChannel(message.ChannelID) {
		var reply = fmt.Sprintf("I do not have command `%s`", args[0])
		state.SendReply(message.ChannelID, reply)
	}
}

func cmdHelp(message *discordgo.MessageCreate, args []string) {
	// Build array of the keys in CmdFuncs
	var keys []string
	for k := range CmdFuncs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build message (sorted by keys) of the commands
	var cmds = "Command notation: \n`" + CmdChar + "[command] [arguments]`\n"
	cmds += "Commands:\n```\n"
	for _, key := range keys {
		cmds += fmt.Sprintf("%s - %s\n", key, CmdFuncs[key].help)
	}
	cmds += "```\n"
	state.SendReply(message.ChannelID, cmds)
}

func cmdVersion(message *discordgo.MessageCreate, args []string) {
	reply := fmt.Sprintf("Version: %v", Version)
	state.SendReply(message.ChannelID, reply)
}

func cmdHere(message *discordgo.MessageCreate, args []string) {
	channel, err := state.Session.Channel(message.ChannelID)
	if err != nil {
		panic(err)
	}

	if state.Channels[channel.ID] {
		state.SendReply(channel.ID, "Yeah")
		return
	}
	state.SendReply(channel.ID, "Fuck on me")
	rate.InitChannel(channel.ID)
}

func cmdNotHere(message *discordgo.MessageCreate, args []string) {
	channel, _ := state.Session.Channel(message.ChannelID)
	state.Channels[channel.ID] = false
	state.Servers[channel.GuildID] = false
	db.AddChannel(channel, false)

}

func cmdStats(message *discordgo.MessageCreate, args []string) {
	leaders, losers := rate.GetRespec()
	var stats = "Leaderboard:\n```\n"
	stats += leaders
	stats += "```"
	stats += "\nLosers:` "
	stats += strings.Join(losers, ", ")
	stats += " `"
	state.SendReply(message.ChannelID, stats)
}

func cmdBet(message *discordgo.MessageCreate, args []string) {
	games.ManualBetCmd(message.Message, args)
}

func cmdCard(message *discordgo.MessageCreate, args []string) {
	card := cards.GenerateCard()
	state.SendReply(message.ChannelID, fmt.Sprintf("%v", card))
}
