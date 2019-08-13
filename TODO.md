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

### Differences

- The "regular" role has essentially been removed. All subs are "regulars", and all "regulars" are subs.
This means that the `subsRegsMinusLinks` option doesn't make much sense. Instead, it may be referred to
as `subsMayLink`, which when true means subs may link in chat.
- Subs are allowed to post links by default, as regulars would be.
- Command lists (`!list`) may be run via repeats and schedules.
- The special `-1`/`admin` mode has been removed; it can be reproduced using another mode.
- When quotes are deleted, quotes after it do not have their numbers changed.
The new `!quote compact <num>` command can be used to compact the quote list and remove
any holes left by deleted quotes. (The same applies to autoreplies.)

### Functionality

- [x] Link detection and moderation
    - [x] YouTube API access to grab URLs
- [x] Twitch API stuff
    - [x] For many things, this requires OAuth, which means the web service.

### Commands

- [ ] General commands
    - [x] Join/part (in bot's channel)
    - [x] Part (in other channels); waiting until this can have "confirm" functionality
    - [x] ~~Topic~~
    - [x] Viewers (*Twitch API*)
    - [x] Chatters (*Twitch API*)
    - [x] Uptime (*Twitch API*)
    - [x] LastFM
        - [x] `music`
        - [x] Song link
    - [ ] Bot help (*Website*)
    - [x] ~~Commercial~~
    - [x] Game (*Twitch API*)
        - [x] Get game
        - [x] Set game
    - [x] Status (*Twitch API*)
        - [x] Get status
        - [x] Set status
    - [ ] statusgame/steamgame (*Steam API*)
    - [x] ~~XBox game~~
    - [ ] Follow me (*Twitch API*)
        - Should this be automatic?
    - [x] ~~Viewer stats~~ (Use twitchtracker or sullygnome)
    - [x] ~~Punish stats~~ Twitch provides some element of this already; could bring back if wanted.
    - [ ] What should I play (*Steam API*)
    - [ ] Google
    - [x] ~~Wiki~~
    - [x] Is live (*Twitch API*)
    - [x] Is here (*Twitch API*)
- [ ] Custom Commands
    - [x] Cooldowns
    - [x] Add
    - [x] Delete
    - [x] Restrict
    - [x] Rename (undocumented)
    - [x] Editor (undocumented)
    - [ ] Link to list of commands (*Website*)
    - [ ] Clone
    - [ ] Automatic restriction based on known actions
- [x] Repeats
    - [x] Add
    - [x] Delete
    - [x] On/off
    - [x] List
    - [x] Actually execute the commands rather than printing them verbatim.
- [x] Schedule
    - [x] Add
    - [x] Delete
    - [x] On/off
    - [x] List
    - [x] Actually execute the commands rather than printing them verbatim.
- [x] Auto-replies
    - [x] Add
    - [x] Remove
    - [x] Edit
    - [x] List
    - [x] Actions inside of autoreply responses
    - [x] Similarly to quotes, emulate old list behavior
- [ ] "Fun"
    - [x] ~~Throw~~ (Use a custom command.)
    - [ ] Winner
        - Expand to pick from only subs
    - [x] ~~Hug~~ (Use a custom command.)
    - [x] Conch/helix (Requires quotes)
    - [ ] Urban
    - [x] ~~Me~~ (Use a custom command.)
    - [x] ~~Race~~
    - [x] XKCD
- [ ] Random/roll
    - [x] ~~`regular`/`sub`~~ The Twitch TMI endpoint doesn't identify users as subs.
    - [x] Cooldowns
    - [x] Integer
    - [x] Dice
    - [x] Default
- [x] Quotes
    - [ ] Link to quotes page on website (*Website*)
    - [x] Add
    - [x] Delete
    - [x] Get
    - [x] Get index
    - [x] Random
    - [x] Search
    - [x] CoeBot would delete a quote and shift quotes after it backwards; HortBot doesn't. There needs to be a command to emulate this behavior.
- [x] ~~Poll~~
- [x] ~~Giveaways~~
- [x] Raffles
    - [x] Raffle
    - [x] Enable/disable
    - [x] Reset
    - [x] Count
    - [x] Winner
- [x] ~~Highlights~~
    - Predates Twitch clips; may still consider reviving.
- [x] ~~Binding of Isaac: Rebirth~~
- [x] Moderation
    - [x] Slow mode on/of
    - [x] Sub mode on/of
    - [x] Ban
    - [x] Timeout
    - [x] Purge
    - [x] Link permit (needs detection)
        - [x] `allow` form
    - [x] Clear chat
- [x] Ignores
    - [x] Add/delete
    - [x] List
- [x] ~~Raids~~ Twitch does this better now
    - May partially implement. Requires the bot account in use to be a channel editor.
    - [x] `host`
    - [x] `unhost`
- [ ] Settings
    - Lots of settings here related to other items.
    - [x] ~~`topic`~~
    - [x] `parseYoutube`
    - [x] `shouldModerate`
    - [x] `roll`
        - [x] ~~`timeoutoncriticalfail`~~
        - [x] `default`
        - [x] `cooldown`
        - [x] `userlevel`
    - [x] ~~`songrequest`~~
    - [x] `extralifeid`
    - [ ] `urban`
    - [x] ~~`gamertag`~~
    - [x] `bullet`
    - [x] `subsRegsMinusLinks` AKA `subsMayLink` (see "differences" above)
    - [x] `cooldown`
    - [x] ~~`throw~~
    - [x] `lastfm`
    - [ ] `steam` (*Steam API*)
    - [x] `mode`
    - [x] ~~`commerciallength`~~
    - [ ] `tweet`
    - [x] `prefix`
    - [x] ~~`emoteset`~~ Doesn't seem to be particularly useful with emotes in tags.
    - [ ] `subscriberalerts`
    - [ ] `resubalerts`
- [x] User levels
    - [x] Add/remove reg/mod/owner
    - [x] List
- [x] Filters
    - [x] Warnings, then timeouts.
    - [x] Me
    - [x] Links
        - [x] Permitted domains (and more)
    - [x] Capitals
    - [x] Banned phrases
    - [x] Symbols
    - [x] Max length
    - [x] Emotes
    - [x] Options
        - [x] `on`/`off`
        - [x] `status`
        - [x] `me`
        - [x] `enablewarnings`
        - [x] `timeoutduration`
        - [x] `displaywarnings`
        - [x] `messagelength`
        - [x] `links`
        - [x] `pd`
        - [x] `banphrase`
        - [x] `caps`
        - [x] `emotes`
        - [x] `symbols`
- [ ] Administration
    - [x] ~~`verboseLogging`~~
    - [ ] `imp`
    - [ ] `+whatprefix`
    - [x] ~~`altsend`~~ There is no "alt" connection, so this doesn't have a meaning (but may if send priorities/rate limiting is added later).
    - [x] ~~`disconnect`~~
    - [x] `admin`
        - [x] `channels`
        - [x] ~~`join`~~
        - [x] ~~`part`~~
        - [x] `block`
        - [x] `unblock`
        - [x] ~~`reconnect`~~
        - [x] ~~`reload`~~
        - [x] `color`
        - [x] ~~`loadfilter`~~ Maybe reintroduce later for another purpose.
        - [x] `spam`
        - [x] `#`
        - [x] ~~`trimchannels`~~
- [x] Variables
    - [x] Set
    - [x] Delete
    - [x] Get
    - [x] Increment
    - [x] Decrement
    - [x] Actions / string replacements (see below)
- [x] Lists
    - [x] Add
    - [x] Delete
    - [x] Restrict
    - [x] Add item
    - [x] Delete item
    - [x] Get
    - [x] Random
    - [x] Actually execute the commands rather than printing them verbatim.
- [ ] Misc undocumented stuff
    - [x] ~~Weird testing commands~~ (Twitch resubs are no longer sent in PRIVMSGs)
    - [ ] Steam game
    - [x] Extra Life stuff
    - [x] ~~`strawpoll`~~
    - [x] ~~`channelID`~~
    - [x] ~~`whisper`~~
    - [x] ~~"wp"~~ (Sorry, go make another bot to do this...)
    - [x] ~~`properties`~~ (This endpoint has been removed.)
    - [x] ~~`songrequest`~~
    - [x] ~~`sendUpdate`~~
    - [ ] Custom commands from another channel (`#<user>/`)
    - [x] ~~`modchan`~~ Use `!set mode`
    - [x] ~~`rejoin`~~

### Actions (string replacements)

- [x] `(_GAME_)` (*Twitch API*)
- [x] `(_GAME_CLEAN_)` (*Twitch API*) - `(_GAME_)` but replace all non-alphanum with `-`
- [x] `(_STATUS_)` (*Twitch API*)
- [x] `(_VIEWERS_)` (*Twitch API*)
- [x] `(_CHATTERS_)` (*Twitch API*)
- [ ] `(_STEAM_PROFILE_)` (*Steam API*)
- [ ] `(_STEAM_GAME_)` (*Steam API*)
- [ ] `(_STEAM_SERVER_)` (*Steam API*)
- [ ] `(_STEAM_STORE_)` (*Steam API*)
- [x] `(_SONG_)`
- [x] `(_SONG_URL_)`
- [x] `(_LAST_SONG_)`
- [ ] `(_BOT_HELP_)` (*Website*)
- [x] `(_USER_)`
- [x] `(_QUOTE_)`
- [x] ~~`(_COMMERCIAL_)`~~
- [x] `(_PARAMETER_)`
- [x] `(_PARAMETER_CAPS_)`
- [x] `(_NUMCHANNELS_)`
- [x] ~~`(_XBOX_GAME_)`~~
- [x] ~~`(_XBOX_PROGRESS_)`~~
- [x] ~~`(_XBOX_GAMERSCORE_)`~~
- [x] `(_ONLINE_CHECK_)` (*Twitch API*)
- [x] `(_SUBMODE_ON_)`
- [x] `(_SUBMODE_OFF_)`
- [ ] `(_GAME_IS_<GAME>_)` (*Twitch API*)
- [ ] `(_GAME_IS_NOT_<GAME>_)` (*Twitch API*)
- [x] `(_HOST_<CHANNEL>_)`
- [x] `(_UNHOST_)`
- [x] `(_RANDOM_<MIN>_<MAX>_)`
- [x] `(_RANDOM_INT_<MIN>_<MAX>_)`
- [x] `(_<COMMANDNAME>_COUNT_)`
- [x] `(_PURGE_)`
- [x] `(_TIMEOUT_)`
- [x] `(_BAN_)`
- [x] `(_SUBMODE_OFF_)`
- [x] `(_SUBMODE_OFF_)`
- [x] `(_REGULARS_ONLY_)`
- [x] `(_COMMAND_<NAME>_)` (for autoreplies)
- [ ] Sub message specific actions
- [ ] `(_TWEET_URL_)`
- [x] `(_EXTRALIFE_AMOUNT_)`
- [x] `DATE`, `TIME`, `TIME24`, `DATETIME`, `DATETIME24`
- [x] `UNTIL`, `UNTILSHORT`, `UNTILLONG`
    - This has been slightly modified to allow "real" RFC3339 timestamps. Existing timestamps were
    done in not-quite-RFC3339 in the Eastern time zone. Also, dates more than ~290 years will no longer
    work (sorry).
- [x] VAR actions
- [x] `(_LIST_<name>_RANDOM_)`
- [x] `(_SILENT_)`
- [x] `(_CHANNEL_URL_)`
- [ ] `(_n_)` ("args")


## New features

- [ ] A new command language (scripting!)
- [ ] GMod integration
- [x] More "add" subcommands for !command, to preset restrictions
- [ ] Extra commands for the new `/delete` feature (to replace purging)
- [ ] HowLongToBeat queries
- [ ] Get a link to game stores other than Steam

### Actions

- [x] `(_DELETE_)` (new)
