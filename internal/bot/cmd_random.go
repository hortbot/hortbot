package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

func cmdRandom(ctx context.Context, s *session, cmd string, args string) error {
	if !s.UserLevel.CanAccessPG(s.Channel.RollLevel) {
		return errNotAuthorized
	}

	if err := s.TryRollCooldown(ctx); err != nil {
		return err
	}

	if err := s.TryCooldown(ctx); err != nil {
		return err
	}

	args, _ = splitSpace(args)
	args = strings.ToLower(args)

	if args == "coin" {
		if s.Deps.Rand.Intn(2) == 0 {
			return s.Reply(ctx, "Heads!")
		}
		return s.Reply(ctx, "Tails!")
	}

	var builder strings.Builder
	builder.WriteString(s.UserDisplay)
	builder.WriteString(" rolled: ")

	var count int
	var maxValue int

	if n, _ := fmt.Sscanf(args, "%dd%d", &count, &maxValue); n == 2 {
		if count > 0 && maxValue > 0 {
			if count > 6 {
				count = 6
			}

			for i := range count {
				if i != 0 {
					builder.WriteString(", ")
				}

				v := s.Deps.Rand.Intn(maxValue) + 1
				builder.WriteString(strconv.Itoa(v))
			}

			return s.Reply(ctx, builder.String())
		}
	}

	if args != "" {
		var err error
		maxValue, err = strconv.Atoi(args)
		if err != nil {
			maxValue = s.Channel.RollDefault
		}
	}

	if maxValue <= 0 {
		maxValue = 20
	}

	v := s.Deps.Rand.Intn(maxValue) + 1
	builder.WriteString(strconv.Itoa(v))

	return s.Reply(ctx, builder.String())
}
