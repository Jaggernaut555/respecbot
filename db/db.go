package db

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/Jaggernaut555/respecbot/logging"

	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
)

type User struct {
	Username string `xorm:"varchar(50) not null unique"`
	Respec   int    `xorm:"default 0"`
	ID       string `xorm:"varchar(50) pk"`
}

type Message struct {
	ID        string    `xorm:"varchar(50) pk"`
	ChannelID string    `xorm:"not null"`
	Content   string    `xorm:"varchar(2000) not null"`
	UserID    string    `xorm:"not null"`
	Respec    int       `xorm:"default 0"`
	Time      time.Time `xorm:"not null"`
}

type Channel struct {
	ID      string `xorm:"varchar(50) pk"`
	GuildID string `xorm:"not null"`
	Active  bool   `xorm:"default 0"`
}

type Reaction struct {
	Content   string    `xorm:"varchar(50) pk"`
	MessageID string    `xorm:"varchar(50) pk"`
	UserID    string    `xorm:"varchar(50) pk"`
	Time      time.Time `xorm:"not null"`
	Removed   time.Time `xorm:"default null"`
}

type Mention struct {
	GiverID    string    `xorm:"varchar(50) pk"`
	ReceiverID string    `xorm:"varchar(50) pk"`
	MessageID  string    `xorm:"varchar(50) pk"`
	Time       time.Time `xorm:"not null"`
	Respec     int       `xorm:"default 0"`
}

type DBBet struct {
	ID        uint64    `xorm:"pk autoincr"`
	ChannelID string    `xorm:"not null"`
	StarterID string    `xorm:"not null"`
	Pot       int       `xorm:"default 0"`
	Time      time.Time `xorm:"not null"`
}

func (*DBBet) TableName() string {
	return "Bet"
}

// ID = Bet.ID, table to hold all users who participated in a bet
type BetUsers struct {
	BetID  uint64 `xorm:"pk"`
	UserID string `xorm:"varchar(50) pk"`
	Bet    int    `xorm:"default 0"`
	Won    int    `xorm:"default 0"`
}

type joinReactionMessage struct {
	Reaction `xorm:"extends"`
	Message  `xorm:"extends"`
}

const ()

var (
	engine *xorm.Engine
)

//Setup Start the database with the given password, purge will delete every entry in the database and exit app
func Setup(dbName, dbUsername, dbPassword string, purge bool) {
	engine = &xorm.Engine{}

	if dbPassword == "" {
		logging.Err("Blank Database Password")
		os.Exit(1)
	}

	if dbName == "" {
		logging.Err("Blank Database Name")
		os.Exit(1)
	}

	if dbUsername == "" {
		logging.Err("Blank Database Username")
		os.Exit(1)
	}

	e, err := xorm.NewEngine("mysql", dbUsername+":"+dbPassword+"@/"+dbName+"?charset=utf8mb4")
	if err != nil {
		logging.Err(err.Error())
		os.Exit(1)
	}

	engine = e

	engine.SetMapper(core.SameMapper{})

	createTables(engine)

	if purge {
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

func createTables(e *xorm.Engine) {
	var err error
	if err = e.Sync2(new(User)); err != nil {
		panic(err)
	}
	if err = e.Sync2(new(Message)); err != nil {
		panic(err)
	}
	if err = e.Sync2(new(Reaction)); err != nil {
		panic(err)
	}
	if err = e.Sync2(new(Mention)); err != nil {
		panic(err)
	}
	if err = e.Sync2(new(Channel)); err != nil {
		panic(err)
	}
	if err = e.Sync2(new(DBBet)); err != nil {
		panic(err)
	}
	if err = e.Sync2(new(BetUsers)); err != nil {
		panic(err)
	}
}

func GetTotalRespec() (total int) {
	var user User

	temp, err := engine.SumInt(user, "Respec")
	if err != nil {
		panic(err)
	}

	return int(temp)
}

func GetUserRespec(discordUser *discordgo.User) (respec int) {
	user := &User{Username: discordUser.String(), ID: discordUser.ID}
	has, err := engine.Get(user)
	if err != nil {
		panic(err)
	}
	if has {
		respec = user.Respec
	}
	return
}

func GetTopUser() (userID string) {
	user := new(User)
	has, err := engine.Table("User").Select("*").Desc("Respec").Get(user)
	if err != nil {
		panic(err)
	}
	if has {
		return user.ID
	}
	return ""
}

func UserIsTop(discordUser *discordgo.User) bool {
	user := new(User)
	has, err := engine.Table("User").Select("*").Desc("Respec").Get(user)
	if err != nil {
		panic(err)
	}
	if has && user.ID == discordUser.ID {
		return true
	}
	return false
}

func GetRulingClass(list *map[string]bool) {
	var users []User
	if err := engine.Find(&users); err != nil {
		panic(err)
	}
	total := float64(GetTotalRespec())
	var pairs pairList
	for _, v := range users {
		pairs = append(pairs, pair{Key: v.ID, Value: v.Respec})
	}
	sort.Sort(sort.Reverse(pairs))
	var totalPercent float64
	for _, v := range pairs {
		if totalPercent > 50 {
			(*list)[v.Key] = false
			continue
		}
		percent := (float64(v.Value) / total) * 100
		totalPercent += percent
		(*list)[v.Key] = true

	}
}

func LoadRespec(list *map[string]int) {
	var users []User
	if err := engine.Find(&users); err != nil {
		panic(err)
	}
	for _, v := range users {
		(*list)[v.Username] = v.Respec
	}
}

func GainRespec(discordUser *discordgo.User, respec int) {
	user := &User{Username: discordUser.String(), ID: discordUser.ID}
	has, err := engine.Get(user)
	if err != nil {
		panic(err)
	}
	if has {
		user.Respec += respec
		if _, err = engine.ID(core.PK{user.ID}).Cols("Respec").Update(user); err != nil {
			panic(err)
		}
	} else {
		user.Respec = respec
		if _, err = engine.Insert(user); err != nil {
			panic(err)
		}
	}
}

func NewMessage(discordUser *discordgo.User, message *discordgo.Message, numRespec int, timeStamp time.Time) {
	msg := &Message{ID: message.ID, Content: message.Content, ChannelID: message.ChannelID, Respec: numRespec, UserID: discordUser.String(), Time: timeStamp}
	if _, err := engine.Insert(msg); err != nil {
		panic(err)
	}
}

func MessageExists(messageID string) (has bool) {
	has, err := engine.Exist(&Message{ID: messageID})
	if err != nil {
		panic(err)
	}
	return
}

func AddChannel(discordChannel *discordgo.Channel, active bool) {
	channel := &Channel{ID: discordChannel.ID, GuildID: discordChannel.GuildID}
	has, err := engine.Get(channel)
	if err != nil {
		panic(err)
	}
	channel.Active = active
	if has {
		if _, err = engine.Id(core.PK{channel.ID}).Cols("Active").Update(channel); err != nil {
			panic(err)
		}
	} else {
		if _, err = engine.Insert(channel); err != nil {
			panic(err)
		}
	}
}

func GetUserLastMessageTime(userID string) (timeStamp time.Time, ok bool) {
	message := Message{UserID: userID}
	has, err := engine.Select("UserId, max(Time) AS Time").GroupBy("UserID").Get(&message)
	if err != nil {
		panic(err)
	}
	if has {
		timeStamp = message.Time
		ok = true
	}
	return
}

func AddMention(giver *discordgo.User, receiver *discordgo.User, message *discordgo.Message, numRespec int, timeStamp time.Time) {
	mention := Mention{GiverID: giver.String(), ReceiverID: receiver.String(), MessageID: message.ID, Respec: numRespec, Time: timeStamp}
	if _, err := engine.Insert(mention); err != nil {
		panic(err)
	}
}

func GetUserLastMentionedTime(userID string) (timeStamp time.Time, ok bool) {
	mention := Mention{ReceiverID: userID}
	has, err := engine.Select("ReceiverID, max(Time) AS Time").GroupBy("ReceiverID").Get(&mention)
	if err != nil {
		panic(err)
	}
	if has {
		timeStamp = mention.Time
		ok = true
	}
	return
}

func ReactionAdd(discordUser *discordgo.User, rctn *discordgo.MessageReaction, timeStamp time.Time) {
	reaction := Reaction{MessageID: rctn.MessageID, UserID: discordUser.String(), Content: rctn.Emoji.ID}

	has, err := engine.Exist(&reaction)
	if err != nil {
		panic(err)
	}

	if has {
		if _, err = engine.Delete(&reaction); err != nil {
			panic(err)
		}
	}

	reaction.Time = timeStamp

	if _, err = engine.Insert(reaction); err != nil {
		panic(err)
	}
}

func GetUserLastReactionAddTime(giverID, receiverID string) (timeStamp time.Time, ok bool) {
	rm := joinReactionMessage{}

	has, err := engine.Table("Reaction").Alias("r").Select("r.UserID, m.UserID, max(r.Time) AS Time").
		Join("INNER", []string{"Message", "m"}, "r.MessageID = m.ID").
		Where("r.Time = (SELECT max(Time) From Reaction b WHERE r.UserID = b.UserID AND m.ID = b.MessageID)").
		And("r.UserID = ?", giverID).And("m.UserID = ?", receiverID).
		GroupBy("r.UserID, m.UserID").
		Get(&rm)
	if err != nil {
		panic(err)
	}
	if has {
		timeStamp = rm.Reaction.Time
		ok = true
	}
	return
}

func ReactionRemove(discordUser *discordgo.User, rctn *discordgo.MessageReaction, timeStamp time.Time) {
	reaction := Reaction{MessageID: rctn.MessageID, UserID: discordUser.String(), Content: rctn.Emoji.ID}

	has, err := engine.Get(&reaction)
	if err != nil {
		panic(err)
	}

	if has {
		if _, err = engine.ID(core.PK{reaction.Content, reaction.MessageID, reaction.UserID}).Cols("Removed").Update(Reaction{Removed: timeStamp}); err != nil {
			panic(err)
		}
	} else {
		reaction.Time = timeStamp
		reaction.Removed = timeStamp
		if _, err = engine.Insert(reaction); err != nil {
			panic(err)
		}
	}
}

func GetUserLastReactionRemoveTime(giverID, receiverID string) (timeStamp time.Time, ok bool) {
	rm := joinReactionMessage{}

	has, err := engine.Table("Reaction").Alias("r").Select("r.UserID, m.UserID, max(r.Removed) AS Removed").
		Join("INNER", []string{"Message", "m"}, "r.MessageID = m.ID").
		Where("r.Removed = (SELECT max(Removed) From Reaction b WHERE r.UserID = b.UserID AND m.ID = b.MessageID)").
		And("r.UserID = ?", giverID).And("m.UserID = ?", receiverID).
		GroupBy("r.UserID, m.UserID").
		Get(&rm)
	if err != nil {
		panic(err)
	}
	if has {
		timeStamp = rm.Reaction.Removed
		ok = true
	}
	return
}

func RecordBet(b DBBet, users []BetUsers) {
	_, err := engine.Table("Bet").Insert(b)
	if err != nil {
		panic(err)
	}
	_, err = engine.Table("Bet").Get(&b)
	if err != nil {
		panic(err)
	}

	for _, v := range users {
		if _, err := engine.Insert(&BetUsers{UserID: v.UserID, BetID: b.ID, Bet: v.Bet, Won: v.Won}); err != nil {
			panic(err)
		}
	}
}

func LoadActiveChannels(chanList *map[string]bool, guildList *map[string]bool) {
	var channels []Channel

	if err := engine.Find(&channels); err != nil {
		panic(err)
	}

	for _, v := range channels {
		if v.Active {
			(*chanList)[v.ID] = v.Active
			(*guildList)[v.GuildID] = v.Active
		}
	}
}

func purgeDB() error {
	engine.ShowSQL(true)
	logging.Log("Purging Database")
	var users []User
	var messages []Message
	var reactions []Reaction
	var mention []Mention
	var channels []Channel
	var dbbet []DBBet
	var betusers []BetUsers
	if err := engine.Find(&users); err != nil {
		return err
	}
	for _, v := range users {
		if _, err := engine.Delete(&v); err != nil {
			return err
		}
	}
	if err := engine.Find(&messages); err != nil {
		return err
	}
	for _, v := range messages {
		if _, err := engine.Delete(&v); err != nil {
			return err
		}
	}
	if err := engine.Find(&reactions); err != nil {
		return err
	}
	for _, v := range reactions {
		if _, err := engine.Delete(&v); err != nil {
			return err
		}
	}
	if err := engine.Find(&mention); err != nil {
		return err
	}
	for _, v := range mention {
		if _, err := engine.Delete(&v); err != nil {
			return err
		}
	}
	if err := engine.Find(&channels); err != nil {
		return err
	}
	for _, v := range channels {
		if _, err := engine.Delete(&v); err != nil {
			return err
		}
	}
	if err := engine.Find(&dbbet); err != nil {
		return err
	}
	for _, v := range dbbet {
		if _, err := engine.Delete(&v); err != nil {
			return err
		}
	}
	if err := engine.Find(&betusers); err != nil {
		return err
	}
	for _, v := range betusers {
		if _, err := engine.Delete(&v); err != nil {
			return err
		}
	}

	return nil
}

type pair struct {
	Key   string
	Value int
}

type pairList []pair

func (p pairList) Len() int           { return len(p) }
func (p pairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p pairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
