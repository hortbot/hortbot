package repeat

import "github.com/robfig/cron/v3"

func ToCron(s cron.Schedule) *Cron {
	return &Cron{expr: s}
}
