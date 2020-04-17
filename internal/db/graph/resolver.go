package graph

import "database/sql"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver resolves GraphQL queries.
type Resolver struct {
	DB *sql.DB // TODO: Don't use a global DB instance, use transactions for the entire query.
}
