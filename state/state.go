package state

import (
	"github.com/Jaggernaut555/respecbot/db"
	"github.com/bwmarrin/discordgo"
)

var (
	Session  *discordgo.Session
	Channels map[string]bool
	Servers  map[string]bool
)

func InitChannels() {
	Channels = map[string]bool{}
	Servers = map[string]bool{}

	db.LoadActiveChannels(&Channels, &Servers)
}

//SendReply Send a reply to the discord session
func SendReply(channelID string, reply string) {
	Session.ChannelMessageSend(channelID, reply)
}

//SendEmbed Send an embed to the discord session
func SendEmbed(channelID string, embed *discordgo.MessageEmbed) (msg *discordgo.Message) {
	msg, _ = Session.ChannelMessageSendEmbed(channelID, embed)
	return
}

func IsValidChannel(channelID string) bool {
	return Channels[channelID]
}
