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
		l AccessLevel
	}{
		{
			s: models.AccessLevelEveryone,
			l: AccessLevelEveryone,
		},
		{
			s: models.AccessLevelSubscriber,
			l: AccessLevelSubscriber,
		},
		{
			s: models.AccessLevelModerator,
			l: AccessLevelModerator,
		},
		{
			s: models.AccessLevelVip,
			l: AccessLevelVIP,
		},
		{
			s: models.AccessLevelBroadcaster,
			l: AccessLevelBroadcaster,
		},
		{
			s: models.AccessLevelAdmin,
			l: AccessLevelAdmin,
		},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			t.Parallel()
			l := newAccessLevel(test.s)
			assert.Equal(t, l, test.l)
			assert.Equal(t, l.PGEnum(), test.s)
		})
	}

	unknown := newAccessLevel("what")
	assert.Equal(t, unknown, AccessLevelUnknown)

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
		user     AccessLevel
		resource AccessLevel
		ok       bool
	}{
		{
			user:     AccessLevelUnknown,
			resource: AccessLevelUnknown,
			ok:       false,
		},
		{
			user:     AccessLevelSuperAdmin,
			resource: AccessLevelUnknown,
			ok:       true,
		},
		{
			user:     AccessLevelUnknown,
			resource: AccessLevelEveryone,
			ok:       false,
		},
		{
			user:     AccessLevelEveryone,
			resource: AccessLevelEveryone,
			ok:       true,
		},
		{
			user:     AccessLevelEveryone,
			resource: AccessLevelSubscriber,
			ok:       false,
		},
		{
			user:     AccessLevelAdmin,
			resource: AccessLevelModerator,
			ok:       true,
		},
		{
			user:     AccessLevelSuperAdmin,
			resource: AccessLevelAdmin,
			ok:       true,
		},
		{
			user:     AccessLevelAdmin,
			resource: AccessLevelSuperAdmin,
			ok:       false,
		},
		{
			user:     AccessLevelAdmin + 100, // Hypothetical
			resource: AccessLevelModerator,
			ok:       false,
		},
	}

	for _, test := range tests {
		t.Run(test.user.String()+"-"+test.resource.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.user.CanAccess(test.resource), test.ok)
		})
	}
}
