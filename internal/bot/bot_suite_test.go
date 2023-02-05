package bot_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/connorkuehl/popple/internal/bot"
	"github.com/connorkuehl/popple/internal/command"
	"github.com/connorkuehl/popple/internal/database/sqlite"
	"github.com/connorkuehl/popple/internal/discord"
	"github.com/connorkuehl/popple/internal/discord/discordtest"
	"github.com/connorkuehl/popple/internal/popple"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bot Suite")
}

var _ = Describe("Bot", func() {
	var (
		botName = "popple"
		db      *sqlite.DB
		router  *command.Router
		session *discordtest.ResponseRecorder
	)

	BeforeEach(func() {
		var (
			err     error
			cleanup func()
		)

		router = command.NewRouter(botName)
		db, cleanup, err = sqlite.NewInMemory()
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(cleanup)
	})

	When("the announce command is invoked", func() {
		Context("and it is missing an argument", func() {
			It("responds with an error", func(ctx SpecContext) {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "1234", ChannelID: "9876", Content: fmt.Sprintf("%s announce", botName)},
				})

				b := bot.New(session, db, router)
				_ = b.Listen(ctx)
				Expect(session.Responses).To(ConsistOf(discordtest.Response{Message: discordtest.Message{ChannelID: "9876", Content: `Valid announce settings are "yes", "on", "no", "off"`}}))
			})
		})

		Context("and it has an invalid argument", func() {
			It("responds with an error", func(ctx SpecContext) {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "1234", ChannelID: "9876", Content: fmt.Sprintf("%s announce potato", botName)},
				})

				b := bot.New(session, db, router)
				_ = b.Listen(ctx)
				Expect(session.Responses).To(ConsistOf(discordtest.Response{Message: discordtest.Message{ChannelID: "9876", Content: `Valid announce settings are "yes", "on", "no", "off"`}}))
			})
		})

		Context(`and its value is "yes" or "on"`, Ordered, func() {
			var b *bot.Bot
			var conf1234, conf5678 popple.ServerConfig

			BeforeAll(func() {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "2", GuildID: "1234", ChannelID: "1010", Content: fmt.Sprintf("%s announce yes", botName)},
					{ID: "3", GuildID: "5678", ChannelID: "2020", Content: fmt.Sprintf("%s announce on", botName)},
				})
				b = bot.New(session, db, router)
				_ = b.Listen(context.Background())

				var err error
				conf1234, err = db.Config(context.Background(), "1234")
				Expect(err).ToNot(HaveOccurred())

				conf5678, err = db.Config(context.Background(), "5678")
				Expect(err).ToNot(HaveOccurred())
			})

			It("reacts with an affirmative emoji", func(ctx SpecContext) {
				Expect(session.Responses).Should(ConsistOf([]discordtest.Response{
					{Reaction: discordtest.Reaction{ChannelID: "1010", MessageID: "2", Emoji: "✅"}},
					{Reaction: discordtest.Reaction{ChannelID: "2020", MessageID: "3", Emoji: "✅"}},
				}))
			})

			It("sets no_announce to false in the database", func(ctx SpecContext) {
				Expect(conf1234.NoAnnounce).To(BeFalse())
				Expect(conf5678.NoAnnounce).To(BeFalse())
			})
		})

		Context(`and its value is "off" or "no"`, Ordered, func() {
			var b *bot.Bot
			var conf1234, conf5678 popple.ServerConfig

			BeforeAll(func() {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "2", GuildID: "1234", ChannelID: "1010", Content: fmt.Sprintf("%s announce no", botName)},
					{ID: "3", GuildID: "5678", ChannelID: "2020", Content: fmt.Sprintf("%s announce off", botName)},
				})
				b = bot.New(session, db, router)
				_ = b.Listen(context.Background())

				var err error
				conf1234, err = db.Config(context.Background(), "1234")
				Expect(err).ToNot(HaveOccurred())

				conf5678, err = db.Config(context.Background(), "5678")
				Expect(err).ToNot(HaveOccurred())
			})

			It("reacts with an affirmative emoji", func(ctx SpecContext) {
				Expect(session.Responses).To(ConsistOf([]discordtest.Response{
					{Reaction: discordtest.Reaction{ChannelID: "1010", MessageID: "2", Emoji: "✅"}},
					{Reaction: discordtest.Reaction{ChannelID: "2020", MessageID: "3", Emoji: "✅"}},
				}))
			})

			It("sets no_announce to true in the database", func(ctx SpecContext) {
				Expect(conf1234.NoAnnounce).To(BeTrue())
				Expect(conf5678.NoAnnounce).To(BeTrue())
			})
		})
	})

	When("bumping karma", func() {
		Context("and no karma is bumped", Ordered, func() {
			var board popple.Board

			BeforeAll(func() {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: "hello+ world, hi"},
				})
				b := bot.New(session, db, router)
				_ = b.Listen(context.Background())
			})

			It("does not interact with the channel", func(ctx SpecContext) {
				Expect(session.Responses).To(HaveLen(0))
			})

			It("does not modify the database", func(ctx SpecContext) {
				var err error
				board, err = db.Leaderboard(ctx, "123", 10)
				Expect(err).ToNot(HaveOccurred())

				Expect(board).To(HaveLen(0))
			})
		})

		Context("and a new entity gets bumped", Ordered, func() {
			var saved []popple.Entity

			BeforeAll(func() {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: "popple++"},
				})
				b := bot.New(session, db, router)
				_ = b.Listen(context.Background())

				var err error
				saved, err = db.Entities(context.Background(), "123", "popple")
				Expect(err).ToNot(HaveOccurred())
			})

			It("tells the channel how much karma the entity has", func() {
				Expect(session.Responses).To(ContainElement(discordtest.Response{Message: discordtest.Message{
					ChannelID: "456",
					Content:   "popple has 1 karma.",
				}}))
			})

			It("persists the entity with 1 karma to the database", func() {
				Expect(saved).To(ContainElement(popple.Entity{Name: "popple", Karma: 1}))
				Expect(saved).To(HaveLen(1))
			})
		})

		Context("and a pre-existing entity's karma is bumped", Ordered, func() {
			var saved []popple.Entity

			BeforeAll(func() {
				err := db.PutEntities(context.Background(), "123", popple.Entity{Name: "panda", Karma: 12})
				Expect(err).ToNot(HaveOccurred())

				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: "beluga panda++ whales"},
				})
				b := bot.New(session, db, router)
				_ = b.Listen(context.Background())

				saved, err = db.Entities(context.Background(), "123", "panda")
				Expect(err).ToNot(HaveOccurred())
			})

			It("tells the channel how much karma the entity has now", func() {
				Expect(session.Responses).To(ContainElement(discordtest.Response{Message: discordtest.Message{
					ChannelID: "456",
					Content:   "panda has 13 karma.",
				}}))
			})

			It("updates the entity's persisted karma", func() {
				Expect(saved).To(ContainElement(popple.Entity{Name: "panda", Karma: 13}))
				Expect(saved).To(HaveLen(1))
			})
		})

		Context("and an entity's karma is bumped but server has muted announcements", Ordered, func() {
			var saved []popple.Entity

			BeforeAll(func() {
				err := db.PutConfig(context.Background(), popple.ServerConfig{ServerID: "123", NoAnnounce: true})
				Expect(err).ToNot(HaveOccurred())

				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: "ganondorf--"},
				})
				b := bot.New(session, db, router)
				_ = b.Listen(context.Background())

				saved, err = db.Entities(context.Background(), "123", "ganondorf")
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not interact with the channel", func(ctx SpecContext) {
				Expect(session.Responses).To(HaveLen(0))
			})

			It("still persists the new karma counts", func(ctx SpecContext) {
				Expect(saved).To(ContainElement(popple.Entity{Name: "ganondorf", Karma: -1}))
				Expect(saved).To(HaveLen(1))
			})
		})
	})

	When("checking karma", func() {
		Context("and no entity names are given", func() {
			It("does not interact with the channel", func(ctx SpecContext) {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: botName + " karma"},
				})
				b := bot.New(session, db, router)
				_ = b.Listen(ctx)

				Expect(session.Responses).To(HaveLen(0))
			})
		})

		Context("and given name is not persisted", func() {
			It("says the entity has zero karma", func(ctx SpecContext) {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: botName + " karma potatopirate"},
				})
				b := bot.New(session, db, router)
				_ = b.Listen(ctx)

				Expect(session.Responses).To(ContainElement(discordtest.Response{Message: discordtest.Message{
					ChannelID: "456",
					Content:   "potatopirate has 0 karma.",
				}}))
			})
		})

		Context("and the given name is persisted", Ordered, func() {
			It("emits the entity's karma count", func(ctx SpecContext) {
				db.PutEntities(ctx, "123", popple.Entity{Name: "mned", Karma: 999})

				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: botName + " karma mned"},
				})
				b := bot.New(session, db, router)
				_ = b.Listen(ctx)

				Expect(session.Responses).To(ContainElement(discordtest.Response{Message: discordtest.Message{
					ChannelID: "456",
					Content:   "mned has 999 karma.",
				}}))
			})
		})
	})

	When("the leaderboard command is invoked", func() {
		Context("with an invalid argument", func() {
			It("responds with an error message", func(ctx SpecContext) {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: botName + " top -1"},
					{ID: "2", GuildID: "123", ChannelID: "456", Content: botName + " top 0"},
					{ID: "3", GuildID: "123", ChannelID: "456", Content: botName + " top asdf"},
				})

				b := bot.New(session, db, router)
				_ = b.Listen(context.Background())

				Expect(session.Responses).To(Equal([]discordtest.Response{
					{Message: discordtest.Message{ChannelID: "456", Content: `Board size must be a positive, non-zero number`}},
					{Message: discordtest.Message{ChannelID: "456", Content: `Board size must be a positive, non-zero number`}},
					{Message: discordtest.Message{ChannelID: "456", Content: `Board size must be a positive, non-zero number`}},
				}))
			})
		})

		Context("with a valid argument", func() {
			limit := 3

			It("constrains the size of the leaderboard to the argument", func(ctx SpecContext) {
				before := command.DefaultLimit
				command.DefaultLimit = uint(limit)
				defer func() {
					command.DefaultLimit = before
				}()

				var preexisting []popple.Entity
				for i := 0; i < limit+1; i++ {
					preexisting = append(preexisting, popple.Entity{Name: strconv.Itoa(i), Karma: int64(i)})
				}

				err := db.PutEntities(ctx, "123", preexisting...)
				Expect(err).ToNot(HaveOccurred())

				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: botName + " top " + strconv.Itoa(limit)},
				})
				b := bot.New(session, db, router)
				_ = b.Listen(ctx)

				Expect(session.Responses).To(HaveLen(1))

				board := parseBoardOutput(session.Responses[0].Message.Content)
				Expect(board).To(HaveLen(limit))
			})
		})

		Context("for a server without any karma bumps", func() {
			It("responds saying no one has bumped any karma", func(ctx SpecContext) {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: botName + " top"},
				})

				b := bot.New(session, db, router)
				_ = b.Listen(ctx)

				Expect(session.Responses).To(ContainElement(discordtest.Response{Message: discordtest.Message{
					ChannelID: "456",
					Content:   `No one has any karma yet.`,
				}}))
			})
		})

		Context("for a server with karma bumps", func() {
			It("responds with a list ordered from most karma to least karma", func(ctx SpecContext) {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: botName + " top"},
				})

				preexisting := []popple.Entity{
					{Name: "Boop", Karma: 10},
					{Name: "Bip", Karma: -10},
					{Name: "Bop", Karma: 100},
				}
				Expect(db.PutEntities(ctx, "123", preexisting...)).ToNot(HaveOccurred())

				b := bot.New(session, db, router)
				_ = b.Listen(ctx)

				Expect(session.Responses).To(HaveLen(1))

				got := parseBoardOutput(session.Responses[0].Message.Content)
				Expect(got).To(Equal([]popple.Entity{
					{Name: "Bop", Karma: 100},
					{Name: "Boop", Karma: 10},
					{Name: "Bip", Karma: -10},
				}))
			})
		})
	})

	When("the loserboard command is invoked", func() {
		Context("with an invalid argument", func() {
			It("responds with an error message", func(ctx SpecContext) {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: botName + " bot -1"},
					{ID: "2", GuildID: "123", ChannelID: "456", Content: botName + " bot 0"},
					{ID: "3", GuildID: "123", ChannelID: "456", Content: botName + " bot asdf"},
				})

				b := bot.New(session, db, router)
				_ = b.Listen(context.Background())

				Expect(session.Responses).To(Equal([]discordtest.Response{
					{Message: discordtest.Message{ChannelID: "456", Content: `Board size must be a positive, non-zero number`}},
					{Message: discordtest.Message{ChannelID: "456", Content: `Board size must be a positive, non-zero number`}},
					{Message: discordtest.Message{ChannelID: "456", Content: `Board size must be a positive, non-zero number`}},
				}))
			})
		})

		Context("with a valid argument", func() {
			limit := 3

			It("constrains the size of the loserboard to the argument", func(ctx SpecContext) {
				before := command.DefaultLimit
				command.DefaultLimit = uint(limit)
				defer func() {
					command.DefaultLimit = before
				}()

				var preexisting []popple.Entity
				for i := 0; i < limit+1; i++ {
					preexisting = append(preexisting, popple.Entity{Name: strconv.Itoa(i), Karma: int64(i)})
				}

				err := db.PutEntities(ctx, "123", preexisting...)
				Expect(err).ToNot(HaveOccurred())

				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: botName + " bot " + strconv.Itoa(limit)},
				})
				b := bot.New(session, db, router)
				_ = b.Listen(ctx)

				Expect(session.Responses).To(HaveLen(1))

				board := parseBoardOutput(session.Responses[0].Message.Content)
				Expect(board).To(HaveLen(limit))
			})
		})

		Context("for a server without any karma bumps", func() {
			It("responds saying no one has bumped any karma", func(ctx SpecContext) {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: botName + " bot"},
				})

				b := bot.New(session, db, router)
				_ = b.Listen(ctx)

				Expect(session.Responses).To(ContainElement(discordtest.Response{Message: discordtest.Message{
					ChannelID: "456",
					Content:   `No one has any karma yet.`,
				}}))
			})
		})

		Context("for a server with karma bumps", func() {
			It("responds with a list ordered from least karma to most karma", func(ctx SpecContext) {
				session = discordtest.NewResponseRecorder([]discord.Message{
					{ID: "1", GuildID: "123", ChannelID: "456", Content: botName + " bot"},
				})

				preexisting := []popple.Entity{
					{Name: "Boop", Karma: 10},
					{Name: "Bip", Karma: -10},
					{Name: "Bop", Karma: 100},
				}
				Expect(db.PutEntities(ctx, "123", preexisting...)).ToNot(HaveOccurred())

				b := bot.New(session, db, router)
				_ = b.Listen(ctx)

				Expect(session.Responses).To(HaveLen(1))

				got := parseBoardOutput(session.Responses[0].Message.Content)
				Expect(got).To(Equal([]popple.Entity{
					{Name: "Bip", Karma: -10},
					{Name: "Boop", Karma: 10},
					{Name: "Bop", Karma: 100},
				}))
			})
		})
	})
})

func parseBoardOutput(s string) []popple.Entity {
	re := regexp.MustCompile(`\*\s+(.+) has (.+) karma.`)
	matches := re.FindAllStringSubmatch(s, -1)

	var entities []popple.Entity

	for _, match := range matches {
		if len(match) != len([]string{"string", "name", "karma"}) {
			panic("failed to parse")
		}

		karma, err := strconv.Atoi(match[2])
		if err != nil {
			panic(err)
		}

		entities = append(entities, popple.Entity{Name: match[1], Karma: int64(karma)})
	}

	return entities
}
