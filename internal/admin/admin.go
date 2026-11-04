// Package admin manages operator privileges.
package admin

import (
	"context"

	"github.com/gentcod/flatmate/internal/database"
)

// IsAdmin reports whether id belongs to an admin. Any error means no.
func IsAdmin(ctx context.Context, q *database.Queries, id int64) bool {
	v, err := q.GetUserAdmin(ctx, id)
	return err == nil && v != 0
}

// SetAdminByHash flips the admin flag for the user with emailHash.
// Unknown emails return sql.ErrNoRows from the lookup.
func SetAdminByHash(ctx context.Context, q *database.Queries, emailHash []byte, on bool) error {
	u, err := q.GetUserByEmailHash(ctx, emailHash)
	if err != nil {
		return err
	}
	var v int64
	if on {
		v = 1
	}
	return q.SetUserAdmin(ctx, database.SetUserAdminParams{IsAdmin: v, ID: u.ID})
}
