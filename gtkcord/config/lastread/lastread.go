package lastread

import (
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/config"
	"github.com/diamondburned/gtkcord3/internal/log"
)

type State struct {
	path  string
	store *store
}

type store struct {
	sync.Mutex `json:"-"`

	// GuildID -> ChannelID; if GuildID == 0 then DM
	Access map[discord.Snowflake]discord.Snowflake
}

func New(file string) State {
	store := &store{
		Access: make(map[discord.Snowflake]discord.Snowflake),
	}

	if err := store.load(file); err != nil {
		log.Errorln("Failed to load config:", err)
	}

	return State{
		path:  file,
		store: store,
	}
}

func (store *store) load(file string) error {
	store.Lock()
	defer store.Unlock()

	return config.UnmarshalFromFile(file, store)
}

func (store *store) save(file string) error {
	store.Lock()
	defer store.Unlock()

	return config.MarshalToFile(file, store)
}

func (s *State) Access(guild discord.Snowflake) discord.Snowflake {
	s.store.Lock()
	defer s.store.Unlock()

	id, _ := s.store.Access[guild]
	return id
}

func (s *State) SetAccess(guild, channel discord.Snowflake) {
	s.store.Lock()
	s.store.Access[guild] = channel
	s.store.Unlock()

	if err := s.store.save(s.path); err != nil {
		log.Errorln("Failed to save config:", err)
	}
}
