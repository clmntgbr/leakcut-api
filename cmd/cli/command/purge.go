package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Tables that must never be truncated by purge.
var purgeKeepTables = map[string]struct{}{
	"users":            {},
	"goose_db_version": {},
}

func NewPurgeCommand() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Truncate all tables except users (and goose_db_version)",
		Long:  "Deletes all rows from application tables. Keeps users and migration history. Requires --yes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("refusing to purge without --yes")
			}

			sqlDB, err := openSQL()
			if err != nil {
				return err
			}
			defer sqlDB.Close()

			rows, err := sqlDB.Query(`
				SELECT tablename
				FROM pg_tables
				WHERE schemaname = 'public'
				ORDER BY tablename
			`)
			if err != nil {
				return fmt.Errorf("list tables: %w", err)
			}
			defer rows.Close()

			var tables []string
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					return fmt.Errorf("scan table name: %w", err)
				}
				if _, keep := purgeKeepTables[name]; keep {
					continue
				}
				tables = append(tables, name)
			}
			if err := rows.Err(); err != nil {
				return fmt.Errorf("iterate tables: %w", err)
			}
			if len(tables) == 0 {
				fmt.Println("Nothing to purge")
				return nil
			}

			quoted := make([]string, len(tables))
			for i, name := range tables {
				quoted[i] = quoteIdent(name)
			}
			stmt := fmt.Sprintf(
				"TRUNCATE TABLE %s RESTART IDENTITY CASCADE",
				strings.Join(quoted, ", "),
			)
			if _, err := sqlDB.Exec(stmt); err != nil {
				return fmt.Errorf("truncate: %w", err)
			}

			fmt.Printf("Purged %d table(s): %s\n", len(tables), strings.Join(tables, ", "))
			fmt.Println("Kept: users, goose_db_version")
			return nil
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm destructive purge")
	return cmd
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
