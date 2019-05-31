# TODO

## The basics

- [x] Single process mode (usable but not final)
    - MVP
    - Need configuration setup
- [ ] Multi process mode
    - Messaging over NSQ
    - Redis for k/v, dedupe, expiration, cooldowns
    - No-downtime upgrades
    - Figure out global prevention
- [ ] Website
    - Needed for OAuth stuff
- [ ] JSON configuration transferer

## CoeBot features

Some features are not currently planned to be ported and have been crossed out.

### Functionality

- [ ] Link detection and moderation
    - [ ] YouTube API access to grab URLs
- [ ] Twitch API stuff
    - [ ] For many things, this requires OAuth, which means the web service.

### Commands

- [ ] General commands
    - [x] Join/part (in bot's channel)
    - [ ] Part (in other channels); waiting until this can have "confirm" functionality
    - [x] ~~Topic~~
    - [ ] Viewers (*Twitch API*)
    - [ ] Chatters (*Twitch API*)
    - [ ] Uptime (*Twitch API*)
    - [ ] LastFM
        - [ ] Music/np
        - [ ] Song link
    - [ ] Bot help (*Website*)
    - [x] ~~Commercial~~
    - [ ] Game (*Twitch API*)
    - [ ] Status (*Twitch API*)
    - [x] ~~statusgame/steamgame~~ (Steam API is restricted)
    - [x] ~~XBox game~~
    - [ ] Follow me (*Twitch API*)
    - [ ] Viewer stats (*Twitch API*)
    - [ ] Punish stats (requires moderation)
    - [x] ~~What should I play~~ (Steam API is restricted)
    - [ ] Google
    - [ ] Wiki
    - [ ] Is live (*Twitch API*)
    - [x] ~~Is here~~ (Requires persisting twitch membership messages)
- [ ] Custom Commands
    - [ ] Cooldowns
    - [x] Add
    - [x] Delete
    - [x] Restrict
    - [x] Rename (undocumented)
    - [x] Editor (undocumented)
    - [ ] Link to list of commands (*Website*)
    - [ ] Clone
- [ ] Repeats
    - This might need to get reworked to prevent statefulness within the bot.
    - [ ] Add
    - [ ] Delete
    - [ ] On/off
    - [ ] List
- [ ] Schedule
    - [ ] Add
    - [ ] Delete
    - [ ] On/off
    - [ ] List
- [ ] Auto-replies
    - This may be reworked to use names for replies instead of indexes.
    - [ ] Add
    - [ ] Remove
    - [ ] Edit
    - [ ] List
- [ ] "Fun"
    - [x] ~~Throw~~ (Use a custom command.)
    - [x] ~~Winner~~ (Requires persisting twitch membership messages)
    - [ ] Random number
    - [x] ~~Hug~~ (Use a custom command.)
    - [ ] Conch/helix (Requires quotes)
    - [ ] Urban
    - [ ] Me
    - [x] ~~Race~~
- [ ] Quotes
    - [ ] Add
    - [ ] Delete
    - [ ] Get
    - [ ] Get index
    - [ ] Random
    - [ ] Search
- [x] ~~Poll~~
- [x] ~~Giveaways~~
- [ ] Raffles
    - [ ] Raffle
    - [ ] Enable/disable
    - [ ] Reset
    - [ ] Count
    - [ ] Winner
- [x] ~~Highlights~~ (Predates twitch clips, may port for old data)
- [x] ~~Binding of Isaac: Rebirth~~
- [ ] Moderation
    - [ ] Slow mode on/of
    - [ ] Sub mode on/of
    - [ ] Ban
    - [ ] Timeout
    - [ ] Purge (add an argument to this to purge the last X messages)
    - [ ] Link permit (needs detection)
    - [ ] Clear chat
- [x] Ignores
    - [x] Add/delete
    - [x] List
- [x] ~~Raids~~ Twitch does this better now
- [ ] Settings
    - Lots of settings here related to other items.
- [x] User levels
    - [x] Add/remove reg/mod/owner
    - [x] List
- [ ] Filters
    - [ ] Links
    - [ ] Capitals
    - [ ] Banned phrases
    - [ ] Symbols
    - [ ] Emotes
- [ ] Administration
- [ ] Variables
    - [ ] Set
    - [ ] Delete
    - [ ] Get
    - [ ] Increment
    - [ ] Decrement
- [ ] Lists
- [ ] Misc undocumented stuff
    - [ ] Roll
    - [ ] Cross channel commands
    - [x] ~~Weird testing commands~~ (Twitch resubs are no longer sent in PRIVMSGs)
    - [ ] Steam game (expand to other stores?)
    - [ ] Extra Life stuff

### Actions (string replacements)

- [ ] `(_GAME_)` (*Twitch API*)
- [ ] `(_STATUS_)` (*Twitch API*)
- [ ] `(_VIEWERS_)` (*Twitch API*)
- [ ] ~~`(_STEAM_GAME_)`~~
- [ ] ~~`(_STEAM_SERVER_)`~~
- [ ] `(_SONG_)`
- [ ] `(_SONG_URL_)`
- [ ] `(_BOT_HELP_)` (*Website*)
- [ ] `(_USER_)`
- [ ] `(_QUOTE_)`
- [ ] ~~`(_COMMERCIAL_)`~~
- [x] `(_PARAMETER_)`
- [x] `(_PARAMETER_CAPS_)`
- [ ] `(_NUMCHANNELS_)`
- [x] ~~`(_XBOX_GAME_)`~~
- [x] ~~`(_XBOX_PROGRESS_)`~~
- [x] ~~`(_XBOX_GAMERSCORE_)`~~
- [ ] `(_ONLINE_CHECK_)`
- [ ] `(_SUBMODE_ON_)`
- [ ] `(_SUBMODE_OFF_)`
- [ ] `(_GAME_IS_<GAME>_)` (*Twitch API*)
- [ ] `(_GAME_IS_NOT_<GAME>_)` (*Twitch API*)
- [ ] `(_HOST_<CHANNEL>_)`
- [ ] `(_UNHOST_)`
- [ ] `(_RANDOM_<MIN>_<MAX>_)`
- [ ] `(_RANDOM_INT_<MIN>_<MAX>_)`
- [ ] `(_<COMMANDNAME>_COUNT_)`
- [ ] `(_PURGE_)`
- [ ] `(_TIMEOUT_)`
- [ ] `(_BAN_)`
- [ ] `(_SUBMODE_OFF_)`
- [ ] `(_SUBMODE_OFF_)`
- [ ] `(_REGULARS_ONLY_)`
- [ ] `(_COMMAND_<NAME>_)` (for autoreplies)
- [ ] Sub message specific actions
- [ ] `(_TWEET_URL_)`
- [ ] `(_EXTRALIFE_AMOUNT_)`
- [ ] DATE/TIME/UNTIL variants
- [ ] VAR actions
- [ ] `(_SILENT_)`


## New features

- [ ] A new command language (scripting!)
- [ ] GMod integration