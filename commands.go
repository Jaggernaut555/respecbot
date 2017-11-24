package main

import (
	"fmt"
	"sort"
	"strings"

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

// InitCmds Initializes the cmds map
func InitCmds() {
	CmdFuncs = CmdFuncsType{
		"help":     CmdFuncHelpType{cmdHelp, "Prints this list", false},
		"lookatme": CmdFuncHelpType{cmdHere, "Fuck off, user", false},
		"fuckoff":  CmdFuncHelpType{cmdNotHere, "Fuck off, bot", true},
		"version":  CmdFuncHelpType{cmdVersion, "Outputs the current bot version", true},
		"stats":    CmdFuncHelpType{cmdStats, "Displays stats about this bot", true},
		"bet":      CmdFuncHelpType{cmdBet, "WHO GONNA WIN? `bet help`", true},
	}
}

func HandleCommand(message *discordgo.MessageCreate, cmd string) {
	args := strings.Split(cmd, " ")
	if len(args) == 0 {
		return
	}
	CmdFuncHelpPair, ok := CmdFuncs[args[0]]

	if ok {
		if !CmdFuncHelpPair.allowedChannelOnly || isValidChannel(message.ChannelID) {
			CmdFuncHelpPair.function(message, args)
		}
	} else if isValidChannel(message.ChannelID) {
		var reply = fmt.Sprintf("I do not have command `%s`", args[0])
		SendReply(message.ChannelID, reply)
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
	SendReply(message.ChannelID, cmds)
}

func cmdVersion(message *discordgo.MessageCreate, args []string) {
	reply := fmt.Sprintf("Version: %v", Version)
	SendReply(message.ChannelID, reply)
}

func cmdHere(message *discordgo.MessageCreate, args []string) {
	channel, err := DiscordSession.Channel(message.ChannelID)
	if err != nil {
		panic(err)
	}

	if Channels[channel.ID] {
		SendReply(channel.ID, "Yeah")
		return
	}
	Channels[channel.ID] = true
	Servers[channel.GuildID] = true
	dbAddChannel(channel, true)
	SendReply(channel.ID, "Fuck on me")
	initLosers(channel.GuildID)
	initTopUsers(channel.GuildID)
}

func cmdNotHere(message *discordgo.MessageCreate, args []string) {
	channel, _ := DiscordSession.Channel(message.ChannelID)
	Channels[channel.ID] = false
	Servers[channel.GuildID] = false
	dbAddChannel(channel, false)

}

func cmdStats(message *discordgo.MessageCreate, args []string) {
	leaders, losers := GetRespec()
	var stats = "Leaderboard:\n```\n"
	stats += leaders
	stats += "```"
	stats += "\nLosers:` "
	stats += strings.Join(losers, ", ")
	stats += " `"
	SendReply(message.ChannelID, stats)
}

func cmdBet(message *discordgo.MessageCreate, args []string) {
	bet(message.Message, args)
}
