package bot

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"

	"github.com/Southclaws/cj/bot/commands"
	"github.com/Southclaws/cj/forum"
	"github.com/Southclaws/cj/storage"
	"github.com/Southclaws/cj/types"
)

// App stores program state
type App struct {
	config        *types.Config
	discordClient *discordgo.Session
	storage       *storage.API
	forum         *forum.ForumClient
	ready         chan bool
	extensions    []Extension
}

// Extension represents an extension to the bot that receives a pointer to the
// storage backend.
type Extension interface {
	Init(*types.Config, *discordgo.Session, *storage.API, *forum.ForumClient) error
	OnMessage(discordgo.Message) error
}

// Start starts the app with the specified config and blocks until fatal error
func Start(config *types.Config) {
	app := App{
		config: config,
	}

	var err error

	app.forum, err = forum.NewForumClient()
	if err != nil {
		logger.Fatal("failed to initialise forum client", zap.Error(err))
	}

	err = app.ConnectDiscord()
	if err != nil {
		logger.Fatal("failed to connect to discord", zap.Error(err))
	}

	app.extensions = []Extension{
		&commands.CommandManager{},
	}

	for _, ex := range app.extensions {
		err = ex.Init(config, app.discordClient, app.storage, app.forum)
		if err != nil {
			logger.Fatal("failed to initialise extension", zap.Error(err))
		}
	}

	app.forum.NewPostAlert("3", func() {
		app.discordClient.ChannelMessageSend(
			config.PrimaryChannel,
			"New Kalcor Post: http://forum.sa-mp.com/search.php?do=finduser&u=3",
		)
	})

	logger.Debug("started with debug logging enabled",
		zap.Any("config", config))

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGKILL)
	<-signals
}
