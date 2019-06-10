package bot

import "github.com/hortbot/hortbot/internal/pkg/rdb"

type RDB struct {
	d  *rdb.DB
	ch string
}

func (r *RDB) LinkPermit(user string, seconds int) error {
	return r.d.Mark(seconds, "link_permit", r.ch, user)
}

func (r *RDB) HasLinkPermit(user string) (permitted bool, err error) {
	return r.d.CheckAndDelete("link_permit", r.ch, user)
}
