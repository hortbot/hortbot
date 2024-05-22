package eventsub_test

import (
	"encoding/json"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"gotest.tools/v3/assert"
)

func TestUnmarshalBroken(t *testing.T) {
	raw := json.RawMessage("{\"metadata\":{\"message_id\":\"epaiOzTgQgKN_us5Wc2QMoxABftosfv-PdEydoAQ0wM=\",\"message_type\":\"notification\",\"message_timestamp\":\"2024-05-22T17:52:11.048754237Z\",\"subscription_type\":\"channel.chat.message\",\"subscription_version\":\"1\"},\"payload\":{\"subscription\":{\"id\":\"073c30f8-dffc-4e90-a140-689cfa1cac43\",\"status\":\"enabled\",\"type\":\"channel.chat.message\",\"version\":\"1\",\"condition\":{\"broadcaster_user_id\":\"39392583\",\"user_id\":\"55056264\"},\"transport\":{\"method\":\"conduit\",\"conduit_id\":\"896f2a0e-5ba9-430c-87ff-edfca4850479\"},\"created_at\":\"2024-05-22T17:15:00.633267495Z\",\"cost\":0},\"event\":{\"broadcaster_user_id\":\"39392583\",\"broadcaster_user_login\":\"ryuquezacotl\",\"broadcaster_user_name\":\"RyuQuezacotl\",\"chatter_user_id\":\"65628971\",\"chatter_user_login\":\"stefandiewaldfee\",\"chatter_user_name\":\"StefanDieWaldfee\",\"message_id\":\"37eeeb2f-7456-4cdc-9842-9c9aed3c29c9\",\"message\":{\"text\":\"i believe in you ryu!\",\"fragments\":[{\"type\":\"text\",\"text\":\"i believe in you ryu!\",\"cheermote\":null,\"emote\":null,\"mention\":null}]},\"color\":\"#8A2BE2\",\"badges\":[{\"set_id\":\"predictions\",\"id\":\"pink-2\",\"info\":\"no\"},{\"set_id\":\"subscriber\",\"id\":\"0\",\"info\":\"1\"},{\"set_id\":\"glitchcon2020\",\"id\":\"1\",\"info\":\"\"}],\"message_type\":\"text\",\"cheer\":null,\"reply\":null,\"channel_points_custom_reward_id\":null,\"channel_points_animation_id\":null}}}")
	var msg eventsub.WebsocketMessage
	err := json.Unmarshal(raw, &msg)
	assert.NilError(t, err)
}
