package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ThankCat/unio-api/internal/config"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeServerAppDB struct {
	rows       []sqlc.ListEnabledProviderAdaptersRow
	queryErr   error
	queryCount int
}

func (db *fakeServerAppDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (db *fakeServerAppDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	db.queryCount++
	if db.queryErr != nil {
		return nil, db.queryErr
	}
	return &fakeServerAppRows{rows: db.rows, index: -1}, nil
}

func (db *fakeServerAppDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return fakeServerAppRow{}
}

func (db *fakeServerAppDB) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
}

type fakeServerAppRow struct{}

func (r fakeServerAppRow) Scan(dest ...any) error {
	return errors.New("fake server app row is not implemented")
}

type fakeServerAppRows struct {
	rows   []sqlc.ListEnabledProviderAdaptersRow
	index  int
	closed bool
}

func (r *fakeServerAppRows) Close() {
	r.closed = true
}

func (r *fakeServerAppRows) Err() error {
	return nil
}

func (r *fakeServerAppRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (r *fakeServerAppRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *fakeServerAppRows) Next() bool {
	if r.closed {
		return false
	}

	r.index++
	if r.index >= len(r.rows) {
		r.Close()
		return false
	}

	return true
}

func (r *fakeServerAppRows) Scan(dest ...any) error {
	if r.index < 0 || r.index >= len(r.rows) {
		return errors.New("fake server app rows scan called without current row")
	}
	if len(dest) != 3 {
		return fmt.Errorf("expected 3 scan destinations, got %d", len(dest))
	}

	id, ok := dest[0].(*int64)
	if !ok {
		return fmt.Errorf("expected destination 0 to be *int64, got %T", dest[0])
	}
	slug, ok := dest[1].(*string)
	if !ok {
		return fmt.Errorf("expected destination 1 to be *string, got %T", dest[1])
	}
	adapter, ok := dest[2].(*string)
	if !ok {
		return fmt.Errorf("expected destination 2 to be *string, got %T", dest[2])
	}

	row := r.rows[r.index]
	*id = row.ID
	*slug = row.Slug
	*adapter = row.Adapter
	return nil
}

func (r *fakeServerAppRows) Values() ([]any, error) {
	if r.index < 0 || r.index >= len(r.rows) {
		return nil, errors.New("fake server app rows values called without current row")
	}

	row := r.rows[r.index]
	return []any{row.ID, row.Slug, row.Adapter}, nil
}

func (r *fakeServerAppRows) RawValues() [][]byte {
	return nil
}

func (r *fakeServerAppRows) Conn() *pgx.Conn {
	return nil
}

func TestNewServerAppBuildsHandlerAfterProviderPreflight(t *testing.T) {
	db := &fakeServerAppDB{
		rows: []sqlc.ListEnabledProviderAdaptersRow{
			{ID: 1, Slug: "openai", Adapter: "openai"},
		},
	}

	app, err := NewServerApp(context.Background(), ServerAppDeps{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Config: newServerAppTestConfig(),
		DB:     db,
	})
	if err != nil {
		t.Fatalf("NewServerApp returned error: %v", err)
	}
	if app == nil || app.Handler == nil {
		t.Fatal("expected server app with handler")
	}
	if db.queryCount != 1 {
		t.Fatalf("expected provider adapter preflight query once, got %d", db.queryCount)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	app.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %q", http.StatusOK, rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected health response body %q", rec.Body.String())
	}
}

func TestNewServerAppReturnsProviderAdapterPreflightError(t *testing.T) {
	db := &fakeServerAppDB{
		rows: []sqlc.ListEnabledProviderAdaptersRow{
			{ID: 1, Slug: "unknown", Adapter: "unknown"},
		},
	}

	app, err := NewServerApp(context.Background(), ServerAppDeps{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Config: newServerAppTestConfig(),
		DB:     db,
	})
	if err == nil {
		t.Fatal("expected preflight error")
	}
	if app != nil {
		t.Fatal("expected no server app when preflight fails")
	}
	if !errors.Is(err, ErrProviderAdapterCapabilityMissing) {
		t.Fatalf("expected ErrProviderAdapterCapabilityMissing, got %v", err)
	}
}

func newServerAppTestConfig() config.Config {
	return config.Config{
		Redis: config.RedisConfig{KeyNamespace: "unio:test"},
		RateLimit: config.RateLimitConfig{
			DefaultLimit:  60,
			DefaultWindow: time.Minute,
			FailurePolicy: "fail_closed",
		},
	}
}
