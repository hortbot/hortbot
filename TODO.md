# TODO

## The basics

- [ ] Single process mode
    - MVP
    - Need configuration setup
- [ ] Multi process mode
    - Messaging over NSQ
    - Redis for k/v, dedupe, expiration, cooldowns
    - No-downtime upgrades
    - Figure out global prevention
- [ ] Website
    - Needed for OAuth stuff

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
    - [ ] Viewers
    - [ ] Chatters
    - [ ] Uptime
    - [ ] LastFM
        - [ ] Music/np
        - [ ] Song link
    - [ ] Bot help
    - [x] ~~Commercial~~
    - [ ] Game
    - [ ] Status
    - [x] ~~statusgame/steamgame~~ (Steam API is restricted)
    - [x] ~~XBox game~~
    - [ ] Follow me
    - [ ] Viewer stats
    - [ ] Punish stats
    - [x] ~~What should I play~~ (Steam API is restricted)
    - [ ] Google
    - [ ] Wiki
    - [ ] Is live
    - [ ] Is here
- [ ] Custom Commands
    - [ ] Cooldowns
    - [x] Add
    - [x] Delete
    - [x] Restrict
    - [x] Rename (undocumented)
    - [x] Editor (undocumented)
    - [ ] Link to list of commands
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
    - [ ] Throw
    - [x] ~~Winner~~
    - [ ] Random number
    - [ ] Hug
    - [ ] Conch/helix
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
- [ ] Ignores
    - [ ] Add/delete
    - [ ] List
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

- [ ] `(_GAME_)`
- [ ] `(_STATUS_)`
- [ ] `(_VIEWERS_)`
- [ ] `(_STEAM_GAME_)`
- [ ] `(_STEAM_SERVER_)`
- [ ] `(_SONG_)`
- [ ] `(_SONG_URL_)`
- [ ] `(_BOT_HELP_)`
- [ ] `(_USER_)`
- [ ] `(_QUOTE_)`
- [ ] `(_COMMERCIAL_)`
- [x] `(_PARAMETER_)`
- [x] `(_PARAMETER_CAPS_)`
- [ ] `(_NUMCHANNELS_)`
- [x] ~~`(_XBOX_GAME_)`~~
- [x] ~~`(_XBOX_PROGRESS_)`~~
- [x] ~~`(_XBOX_GAMERSCORE_)`~~
- [ ] `(_ONLINE_CHECK_)`
- [ ] `(_SUBMODE_ON_)`
- [ ] `(_SUBMODE_OFF_)`
- [ ] `(_GAME_IS_<GAME>_)`
- [ ] `(_GAME_IS_NOT_<GAME>_)`
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
