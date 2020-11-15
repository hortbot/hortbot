package bot

import (
	"testing"

	"github.com/hortbot/hortbot/internal/db/models"
	"gotest.tools/v3/assert"
)

func TestAccessLevelConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		s string
		l accessLevel
	}{
		{
			s: models.AccessLevelEveryone,
			l: levelEveryone,
		},
		{
			s: models.AccessLevelSubscriber,
			l: levelSubscriber,
		},
		{
			s: models.AccessLevelModerator,
			l: levelModerator,
		},
		{
			s: models.AccessLevelVip,
			l: levelVIP,
		},
		{
			s: models.AccessLevelBroadcaster,
			l: levelBroadcaster,
		},
		{
			s: models.AccessLevelAdmin,
			l: levelAdmin,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.s, func(t *testing.T) {
			t.Parallel()
			l := newAccessLevel(test.s)
			assert.Equal(t, l, test.l)
			assert.Equal(t, l.PGEnum(), test.s)
		})
	}

	unknown := newAccessLevel("what")
	assert.Equal(t, unknown, levelUnknown)

	panicked := false

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		unknown.PGEnum()
	}()

	assert.Assert(t, panicked)
}

func TestAccessLevelCanAccess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		user     accessLevel
		resource accessLevel
		ok       bool
	}{
		{
			user:     levelUnknown,
			resource: levelUnknown,
			ok:       false,
		},
		{
			user:     levelSuperAdmin,
			resource: levelUnknown,
			ok:       true,
		},
		{
			user:     levelUnknown,
			resource: levelEveryone,
			ok:       false,
		},
		{
			user:     levelEveryone,
			resource: levelEveryone,
			ok:       true,
		},
		{
			user:     levelEveryone,
			resource: levelSubscriber,
			ok:       false,
		},
		{
			user:     levelAdmin,
			resource: levelModerator,
			ok:       true,
		},
		{
			user:     levelSuperAdmin,
			resource: levelAdmin,
			ok:       true,
		},
		{
			user:     levelAdmin,
			resource: levelSuperAdmin,
			ok:       false,
		},
		{
			user:     levelAdmin + 100, // Hypothetical
			resource: levelModerator,
			ok:       false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.user.String()+"-"+test.resource.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.user.CanAccess(test.resource), test.ok)
		})
	}
}
