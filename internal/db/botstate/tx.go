package botstate

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/aarondl/sqlboiler/v4/boil"
)

// botstateLockClass namespaces botstate's transaction advisory locks.
const botstateLockClass int32 = 0x6B767374

func withKeyLock(ctx context.Context, exec boil.ContextExecutor, namespace, key string) error {
	h := fnv.New32a()
	_, _ = h.Write([]byte(namespace))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(key))

	objid := int32(h.Sum32()) //nolint:gosec // bit pattern preserved; collision space is per-classid.
	if _, err := exec.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock($1, $2)`, botstateLockClass, objid,
	); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	return nil
}
