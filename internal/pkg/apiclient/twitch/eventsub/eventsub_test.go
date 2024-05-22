package eventsub_test

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"gotest.tools/v3/assert"
)

func TestUnmarshalBroken(t *testing.T) {
	tests := []string{
		"{\"metadata\":{\"message_id\":\"epaiOzTgQgKN_us5Wc2QMoxABftosfv-PdEydoAQ0wM=\",\"message_type\":\"notification\",\"message_timestamp\":\"2024-05-22T17:52:11.048754237Z\",\"subscription_type\":\"channel.chat.message\",\"subscription_version\":\"1\"},\"payload\":{\"subscription\":{\"id\":\"073c30f8-dffc-4e90-a140-689cfa1cac43\",\"status\":\"enabled\",\"type\":\"channel.chat.message\",\"version\":\"1\",\"condition\":{\"broadcaster_user_id\":\"39392583\",\"user_id\":\"55056264\"},\"transport\":{\"method\":\"conduit\",\"conduit_id\":\"896f2a0e-5ba9-430c-87ff-edfca4850479\"},\"created_at\":\"2024-05-22T17:15:00.633267495Z\",\"cost\":0},\"event\":{\"broadcaster_user_id\":\"39392583\",\"broadcaster_user_login\":\"ryuquezacotl\",\"broadcaster_user_name\":\"RyuQuezacotl\",\"chatter_user_id\":\"65628971\",\"chatter_user_login\":\"stefandiewaldfee\",\"chatter_user_name\":\"StefanDieWaldfee\",\"message_id\":\"37eeeb2f-7456-4cdc-9842-9c9aed3c29c9\",\"message\":{\"text\":\"i believe in you ryu!\",\"fragments\":[{\"type\":\"text\",\"text\":\"i believe in you ryu!\",\"cheermote\":null,\"emote\":null,\"mention\":null}]},\"color\":\"#8A2BE2\",\"badges\":[{\"set_id\":\"predictions\",\"id\":\"pink-2\",\"info\":\"no\"},{\"set_id\":\"subscriber\",\"id\":\"0\",\"info\":\"1\"},{\"set_id\":\"glitchcon2020\",\"id\":\"1\",\"info\":\"\"}],\"message_type\":\"text\",\"cheer\":null,\"reply\":null,\"channel_points_custom_reward_id\":null,\"channel_points_animation_id\":null}}}",
		"{\"metadata\":{\"message_id\":\"JIHE-SmoEJJvRapt3OPpEb6Iq1gcM_kl3rMTuFCebbM=\",\"message_type\":\"notification\",\"message_timestamp\":\"2024-05-22T20:44:10.044186211Z\",\"subscription_type\":\"channel.chat.message\",\"subscription_version\":\"1\"},\"payload\":{\"subscription\":{\"id\":\"5c32f8d4-163d-4200-bfff-043b55a5ef9c\",\"status\":\"enabled\",\"type\":\"channel.chat.message\",\"version\":\"1\",\"condition\":{\"broadcaster_user_id\":\"22186976\",\"user_id\":\"55056264\"},\"transport\":{\"method\":\"conduit\",\"conduit_id\":\"896f2a0e-5ba9-430c-87ff-edfca4850479\"},\"created_at\":\"2024-05-22T17:14:35.330453713Z\",\"cost\":0},\"event\":{\"broadcaster_user_id\":\"22186976\",\"broadcaster_user_login\":\"hcjustin\",\"broadcaster_user_name\":\"HCJustin\",\"chatter_user_id\":\"55056264\",\"chatter_user_login\":\"coebot\",\"chatter_user_name\":\"CoeBot\",\"message_id\":\"8017b008-4a83-4c79-a7bc-b13ebafd4573\",\"message\":{\"text\":\"coebotBot Join the Slub Club TODAY  hcjHH  and get a FREE GUN  hcjGun  http://subs.twitch.tv/hcjustin\",\"fragments\":[{\"type\":\"emote\",\"text\":\"coebotBot\",\"cheermote\":null,\"emote\":{\"id\":\"793340\",\"emote_set_id\":\"580501\",\"owner_id\":\"55056264\",\"format\":[\"static\"]},\"mention\":null},{\"type\":\"text\",\"text\":\" Join the Slub Club TODAY  \",\"cheermote\":null,\"emote\":null,\"mention\":null},{\"type\":\"emote\",\"text\":\"hcjHH\",\"cheermote\":null,\"emote\":{\"id\":\"1205505\",\"emote_set_id\":\"7351\",\"owner_id\":\"22186976\",\"format\":[\"static\"]},\"mention\":null},{\"type\":\"text\",\"text\":\"  and get a FREE GUN  \",\"cheermote\":null,\"emote\":null,\"mention\":null},{\"type\":\"emote\",\"text\":\"hcjGun\",\"cheermote\":null,\"emote\":{\"id\":\"emotesv2_e8ce8ec84c0c4d29b1d9bbf1b3c74c60\",\"emote_set_id\":\"7351\",\"owner_id\":\"22186976\",\"format\":[\"static\"]},\"mention\":null},{\"type\":\"text\",\"text\":\"  http://subs.twitch.tv/hcjustin\",\"cheermote\":null,\"emote\":null,\"mention\":null}]},\"color\":\"#2E8B57\",\"badges\":[{\"set_id\":\"moderator\",\"id\":\"1\",\"info\":\"\"},{\"set_id\":\"founder\",\"id\":\"0\",\"info\":\"117\"}],\"message_type\":\"text\",\"cheer\":null,\"reply\":null,\"channel_points_custom_reward_id\":null,\"channel_points_animation_id\":null}}}",
	}

	for i, raw := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var msg eventsub.WebsocketMessage
			err := json.Unmarshal([]byte(raw), &msg)
			assert.NilError(t, err)
		})
	}
}
