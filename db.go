package main

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
)

type User struct {
	ID     string `xorm:"pk"`
	Respec int    `xorm:"default 0"`
}

type Message struct {
	ID      string    `xorm:"pk"`
	Content string    `xorm:"varchar(2000) not null"`
	UserID  string    `xorm:"not null"`
	Respec  int       `xorm:"default 0"`
	Time    time.Time `xorm:"not null"`
}

type Reaction struct {
	Content   string    `xorm:"pk"`
	MessageID string    `xorm:"pk"`
	UserID    string    `xorm:"pk"`
	Time      time.Time `xorm:"not null"`
	Removed   time.Time
}

type Respec struct {
	ID         uint64    `xorm:"pk autoincr"`
	GiverID    string    `xorm:"not null"`
	ReceiverID string    `xorm:"not null"`
	Time       time.Time `xorm:"not null"`
	Respec     int       `xorm:"default 0"`
}

type Mention struct {
	GiverID    string    `xorm:"pk"`
	ReceiverID string    `xorm:"pk"`
	MessageID  string    `xorm:"pk"`
	Time       time.Time `xorm:"not null"`
	Respec     int       `xorm:"default 0"`
}

var engine *xorm.Engine

func InitDB() {
	engine = &xorm.Engine{}

	e, err := xorm.NewEngine("mysql", dbUser+":"+dbPassword+"@/"+dbName+"?charset=utf8mb4")
	if err != nil {
		panic(err)
	}

	engine = e

	engine.SetMapper(core.SameMapper{})

	createTables(engine)

	fmt.Println("Database running")
}

func createTables(e *xorm.Engine) {
	e.Sync2(new(User))
	e.Sync2(new(Message))
	e.Sync2(new(Reaction))
	e.Sync2(new(Respec))
	e.Sync2(new(Mention))
}

func dbGetTotalRespec() (total int) {
	var user User

	temp, err := engine.SumInt(user, "Respec")
	if err != nil {
		panic(err)
	}

	total = int(temp)

	return
}

func dbGetUserRespec(discordUser *discordgo.User) (respec int) {
	user := &User{ID: discordUser.String()}
	has, err := engine.Get(user)
	if err != nil {
		panic(err)
	}
	if has {
		respec = user.Respec
	}
	return
}

func dbLoadRespec(list *map[string]int) {
	var users []User
	if err := engine.Find(&users); err != nil {
		panic(err)
	}
	for _, v := range users {
		(*list)[v.ID] = v.Respec
	}
}

func dbGainRespec(discordUser *discordgo.User, respec int) {
	user := &User{ID: discordUser.String()}
	has, err := engine.Get(user)
	if err != nil {
		panic(err)
	}
	if has {
		user.Respec += respec
		if _, err = engine.ID(user.ID).Cols("Respec").Update(user); err != nil {
			panic(err)
		}
	} else {
		user.Respec = respec
		if _, err = engine.Insert(user); err != nil {
			panic(err)
		}
	}
}

func dbNewMessage(discordUser *discordgo.User, message *discordgo.MessageCreate, numRespec int, timeStamp time.Time) {
	msg := &Message{ID: message.ID, Content: message.Content, Respec: numRespec, UserID: discordUser.String(), Time: timeStamp}
	if _, err := engine.Insert(msg); err != nil {
		panic(err)
	}
}

func dbGiveRespec(giver *discordgo.User, receiver *discordgo.User, numRespec int, timeStamp time.Time) {
	respec := &Respec{GiverID: giver.String(), ReceiverID: receiver.String(), Respec: numRespec, Time: timeStamp}
	if _, err := engine.Insert(respec); err != nil {
		panic(err)
	}
}

func dbReactionAdd(discordUser *discordgo.User, rctn *discordgo.MessageReactionAdd, timeStamp time.Time) {
	reaction := Reaction{MessageID: rctn.MessageID, UserID: discordUser.String(), Time: timeStamp, Content: rctn.Emoji.ID}

	fmt.Println(reaction)

	has, err := engine.Get(&reaction)
	if err != nil {
		panic(err)
	}

	if has {
		if _, err = engine.Delete(&reaction); err != nil {
			panic(err)
		}
	}

	if _, err = engine.Insert(reaction); err != nil {
		panic(err)
	}
}

func dbReactionRemove(discordUser *discordgo.User, rctn *discordgo.MessageReactionRemove, timeStamp time.Time) {
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

func dbMention(giver *discordgo.User, receiver *discordgo.User, message *discordgo.Message, numRespec int, timeStamp time.Time) {
	mention := Mention{GiverID: giver.String(), ReceiverID: receiver.String(), MessageID: message.ID, Respec: numRespec, Time: timeStamp}
	if _, err := engine.Insert(mention); err != nil {
		panic(err)
	}
}

func purgeDB() error {
	engine.ShowSQL(true)
	var users []User
	var messages []Message
	var reactions []Reaction
	var respecs []Respec
	var Mention []Mention
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
	if err := engine.Find(&respecs); err != nil {
		return err
	}
	for _, v := range respecs {
		if _, err := engine.Delete(&v); err != nil {
			return err
		}
	}
	if err := engine.Find(&Mention); err != nil {
		return err
	}
	for _, v := range Mention {
		if _, err := engine.Delete(&v); err != nil {
			return err
		}
	}
	return nil

}
