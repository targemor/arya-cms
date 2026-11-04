// Package contenttype implements the Postgres side of user-defined content types: the DDL that
// keeps each content table in line with its definition.
package contenttype

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
	"github.com/lucimux/arya/pkg/postgres"
)

// _tablePrefix namespaces content tables so a definition can never touch an application table
// such as users or schema_migrations.
const _tablePrefix = "content_"

// Repo -.
type Repo struct {
	*postgres.Postgres
}

// New returns a ContentType migrator instrumented with OpenTelemetry tracing spans.
func New(pg *postgres.Postgres) repo.ContentTypeMigrator {
	return newTraced(&Repo{pg})
}

// Migrate creates the table if needed and adds any missing columns, in one transaction. It never
// drops or alters existing columns.
func (r *Repo) Migrate(ctx context.Context, projectSlug string, ct entity.ContentType) error {
	stmts, err := buildDDL(projectSlug, ct)
	if err != nil {
		return fmt.Errorf("ContentTypeRepo - Migrate - buildDDL: %w", err)
	}

	err = pgx.BeginFunc(ctx, r.Pool, func(tx pgx.Tx) error {
		for _, stmt := range stmts {
			if _, err := tx.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("tx.Exec: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("ContentTypeRepo - Migrate - pgx.BeginFunc: %w", err)
	}

	return nil
}

// TableName returns the quoted physical table for a content type within a project.
func TableName(projectSlug, tableName string) string {
	if projectSlug == "" {
		projectSlug = "default"
	}

	return pgx.Identifier{_tablePrefix + projectSlug + "_" + tableName}.Sanitize()
}

// buildDDL renders the statements for ct. Identifiers are validated by entity.ContentType and
// quoted here as well, so nothing user-supplied reaches the SQL unquoted.
func buildDDL(projectSlug string, ct entity.ContentType) ([]string, error) {
	table := TableName(projectSlug, ct.TableName)

	var b strings.Builder

	b.WriteString("CREATE TABLE IF NOT EXISTS ")
	b.WriteString(table)
	b.WriteString(" (id UUID PRIMARY KEY DEFAULT gen_random_uuid(),")
	b.WriteString(" created_at TIMESTAMPTZ NOT NULL DEFAULT now(),")
	b.WriteString(" updated_at TIMESTAMPTZ NOT NULL DEFAULT now())")

	stmts := []string{b.String()}

	if len(ct.Attributes) == 0 {
		return stmts, nil
	}

	b.Reset()
	b.WriteString("ALTER TABLE ")
	b.WriteString(table)

	for i, a := range ct.Attributes {
		colType, err := columnType(a.Type)
		if err != nil {
			return nil, err
		}

		if i > 0 {
			b.WriteByte(',')
		}

		b.WriteString(" ADD COLUMN IF NOT EXISTS ")
		b.WriteString(pgx.Identifier{a.Name}.Sanitize())
		b.WriteByte(' ')
		b.WriteString(colType)
	}

	return append(stmts, b.String()), nil
}

// columnType maps an attribute type to its column type. Columns are nullable: "required" is
// enforced on the payload, since NOT NULL cannot be added to a table that already has rows.
// A media column references the Media Library; deleting the file clears the reference.
func columnType(t entity.AttributeType) (string, error) {
	switch t {
	case entity.AttributeTypeString:
		return "TEXT", nil
	case entity.AttributeTypeInteger:
		return "BIGINT", nil
	case entity.AttributeTypeBoolean:
		return "BOOLEAN", nil
	case entity.AttributeTypeMedia:
		return "UUID REFERENCES upload_files (id) ON DELETE SET NULL", nil
	case entity.AttributeTypeComponent:
		return "JSONB", nil
	default:
		return "", fmt.Errorf("%w: unsupported attribute type %q", entity.ErrInvalidContentType, t)
	}
}
