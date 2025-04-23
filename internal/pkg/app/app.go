package app

import (
	"errors"
	"github.com/bwmarrin/discordgo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"shakalizator/internal/app/config"
	"shakalizator/internal/app/models"
	"shakalizator/internal/app/repository"
	"shakalizator/pkg/logger"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type App struct {
	log *logger.Logger
	cfg *config.Config
	db  *gorm.DB
	bot *discordgo.Session

	usersRepo  *repository.UsersRepository
	videosRepo *repository.VideosRepository
}

func New() error {
	a := &App{
		log: logger.New(),
	}

	var err error
	a.cfg, err = config.NewConfig()
	if err != nil {
		a.log.Error("Error loading config from env", err)
		return err
	}
	a.log.SetLogLevel(a.cfg.LoggerLevel)

	a.db, err = gorm.Open(sqlite.Open("discordbot.db"), &gorm.Config{})
	if err != nil {
		a.log.Error("Failed to open database", err)
		return err
	}
	err = a.db.AutoMigrate(models.Video{}, models.User{})
	if err != nil {
		a.log.Error("Failed to auto-migrate models", err)
		return err
	}

	a.usersRepo = repository.NewUsers(a.log, a.db)
	a.videosRepo = repository.NewVideos(a.log, a.db)

	return RunBot(a)
}

func RunBot(a *App) error {
	var err error
	a.bot, err = discordgo.New("Bot " + a.cfg.BotToken)
	if err != nil {
		a.log.Error("Failed to create discord session", err)
		return err
	}
	a.bot.Identify.Intents |= discordgo.IntentMessageContent
	a.bot.AddHandler(a.processMessage)

	err = a.bot.Open()
	if err != nil {
		a.log.Error("Failed to open discord connection", err)
		return err
	}
	defer a.bot.Close()

	a.log.Info("Bot started successfully")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc

	a.log.Info("Shutting down bot")
	return nil
}

func (a *App) processMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if a.cfg.ChannelID == 0 {
		a.log.Info("ChannelID detected", slog.String("channel_id", m.ChannelID), slog.String("message", m.Content))
		return
	}

	if strconv.FormatUint(a.cfg.ChannelID, 10) != m.ChannelID {
		return
	}

	messageContent := strings.ToLower(m.Content)
	re := regexp.MustCompile(`(?i)https?://(?:www\.)?(?:youtube\.com/(?:watch\?v=|shorts/)|youtu\.be/)([\w-]+)`)
	matches := re.FindStringSubmatch(messageContent)
	if len(matches) != 2 {
		return
	}

	userID, err := strconv.ParseUint(m.Author.ID, 10, 64)
	if err != nil {
		a.log.Error("Error parsing user ID", err, slog.String("authorID", m.Author.ID), slog.String("messageID", m.ID))
		return
	}

	violations, err := a.usersRepo.GetViolations(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.log.Error("Error getting user violations", err, slog.Uint64("userID", userID))
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		a.log.Debug("User not found, creating new entry", slog.Uint64("userID", userID))
		err = a.usersRepo.Create(&models.User{
			ID:         userID,
			Violations: 0,
		})
		if err != nil {
			a.log.Error("Failed to create user", err, slog.Uint64("userID", userID))
			return
		}
	}

	videoID, err := strconv.ParseUint(m.ID, 10, 64)
	if err != nil {
		a.log.Error("Error parsing message ID", err, slog.String("messageID", m.ID))
		return
	}

	video := models.Video{
		ID:     videoID,
		UserID: userID,
		URL:    matches[1],
	}

	v, err := a.videosRepo.Get(video.UserID, video.URL)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.log.Error("Error getting video", err, slog.Uint64("userID", userID), slog.String("url", video.URL))
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		a.log.Debug("Video not found, creating new entry", slog.Uint64("userID", userID), slog.String("url", video.URL))

		err = a.videosRepo.Create(&video)
		if err != nil {
			a.log.Error("Failed to create video", err, slog.Uint64("userID", userID), slog.String("url", video.URL))
			return
		}
	}

	if v != nil {
		a.log.Warn("Duplicate video detected", slog.Uint64("userID", userID), slog.String("url", video.URL), slog.Int("previous_violations", violations))

		err := a.usersRepo.IncrementViolations(userID)
		if err != nil {
			a.log.Error("failed to increment violations", err, slog.Uint64("userID", userID))
			return
		}
		violations++

		err = s.ChannelMessageDelete(m.ChannelID, m.ID)
		if err != nil {
			a.log.Error("Failed to delete message", err, slog.String("channelID", m.ChannelID), slog.String("messageID", m.ID))
		} else {
			a.log.Info("Message deleted", slog.String("author", m.Author.Username), slog.String("content", m.Content), slog.String("channelID", m.ChannelID))
		}

		switch {
		case violations == 1:
			t := time.Now().Add(3 * time.Hour)
			err = s.GuildMemberTimeout(m.GuildID, m.Author.ID, &t)
			if err != nil {
				a.log.Error("Failed to timeout user", err, slog.String("guildID", m.GuildID), slog.Uint64("userID", userID), slog.Duration("duration", 3*time.Hour))
				return
			}
			a.log.Warn("User timed out", slog.Uint64("userID", userID), slog.Duration("duration", 3*time.Hour))
		case violations == 2:
			err = s.GuildMemberDeleteWithReason(m.GuildID, m.Author.ID, "Не спамь одним и тем же роликом! В следующий раз получишь бан")
			if err != nil {
				a.log.Error("Failed to kick user", err, slog.String("guildID", m.GuildID), slog.Uint64("userID", userID))
				return
			}
			a.log.Warn("User kicked", slog.Uint64("userID", userID))
		case violations == 3:
			err = s.GuildBanCreateWithReason(m.GuildID, m.Author.ID, "Не спамь одним и тем же роликом!", 7)
			if err != nil {
				a.log.Error("Failed to ban user", err, slog.String("guildID", m.GuildID), slog.Uint64("userID", userID))
				return
			}
			a.log.Warn("User banned", slog.Uint64("userID", userID), slog.Int("ban_days", 7))
			fallthrough
		case violations > 3:
			err := a.usersRepo.ResetViolations(userID)
			if err != nil {
				a.log.Error("Failed to reset user violations", err, slog.Uint64("userID", userID))
				return
			}

			err = a.videosRepo.Delete(video.URL)
			if err != nil {
				a.log.Error("Failed to delete video", err, slog.Uint64("userID", userID))
				return
			}
		}
	}
}
