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
type CmdFuncType func(*discordgo.Session, *discordgo.MessageCreate, []string)

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
		"respec":   CmdFuncHelpType{cmdRespec, "RESPEC", true},
		"norespec": CmdFuncHelpType{cmdNoRespec, "NO RESPEC", true},
		"stats":    CmdFuncHelpType{cmdStats, "Displays stats about this bot", true},
	}
}

func HandleCommand(session *discordgo.Session, message *discordgo.MessageCreate, cmd string) {
	args := strings.Split(cmd, " ")
	if len(args) == 0 {
		return
	}
	CmdFuncHelpPair, ok := CmdFuncs[args[0]]

	if ok {
		if !CmdFuncHelpPair.allowedChannelOnly || isValidChannel(session, message.ChannelID) {
			CmdFuncHelpPair.function(session, message, args)
		}
	} else if isValidChannel(session, message.ChannelID) {
		var reply = fmt.Sprintf("I do not have command `%s`", args[0])
		SendReply(session, message.ChannelID, reply)
	}
}

func cmdHelp(session *discordgo.Session, message *discordgo.MessageCreate, args []string) {
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
	SendReply(session, message.ChannelID, cmds)
}

func cmdVersion(session *discordgo.Session, message *discordgo.MessageCreate, args []string) {
	SendReply(session, message.ChannelID, "Version: "+Version)
}

func cmdHere(session *discordgo.Session, message *discordgo.MessageCreate, args []string) {
	channel, _ := session.Channel(message.ChannelID)
	if Channels[channel.ID] {
		SendReply(session, channel.ID, "Yeah")
		return
	}
	Channels[channel.ID] = true
	Servers[channel.GuildID] = true
	SendReply(session, channel.ID, "Fuck on me")
}

func cmdNotHere(session *discordgo.Session, message *discordgo.MessageCreate, args []string) {
	channel, _ := session.Channel(message.ChannelID)
	Channels[channel.ID] = false
	Servers[channel.GuildID] = false

}

func cmdStats(session *discordgo.Session, message *discordgo.MessageCreate, args []string) {
	var stats = "Stats:\n```\n"
	stats += GetMostRespec()
	stats += "```"
	SendReply(session, message.ChannelID, stats)
	//SaveRatings()
}

func cmdRespec(session *discordgo.Session, message *discordgo.MessageCreate, args []string) {
	// give a user respec
	GiveRespec(message)
}

func cmdNoRespec(session *discordgo.Session, message *discordgo.MessageCreate, args []string) {
	// lose a user respec
	NoRespec(message)
}
