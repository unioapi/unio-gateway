package apikey

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThankCat/unio-api/internal/platform/store/sqlc"
	"github.com/jackc/pgx/v5"
)

type fakeStore struct {
	arg           sqlc.CreateAPIKeyParams
	called        bool
	err           error
	projectArg    sqlc.GetProjectForUserParams
	projectCalled bool
	projectErr    error
}

func (s *fakeStore) CreateAPIKey(ctx context.Context, arg sqlc.CreateAPIKeyParams) (sqlc.ApiKey, error) {
	s.arg = arg
	s.called = true
	if s.err != nil {
		return sqlc.ApiKey{}, s.err
	}

	return sqlc.ApiKey{
		ID:        1,
		ProjectID: arg.ProjectID,
		Name:      arg.Name,
		KeyPrefix: arg.KeyPrefix,
		ExpiresAt: arg.ExpiresAt,
	}, nil
}

func (s *fakeStore) GetProjectForUser(ctx context.Context, arg sqlc.GetProjectForUserParams) (sqlc.Project, error) {
	s.projectArg = arg
	s.projectCalled = true
	if s.projectErr != nil {
		return sqlc.Project{}, s.projectErr
	}

	return sqlc.Project{
		ID:     arg.ProjectID,
		UserID: arg.UserID,
		Name:   "test project",
	}, nil
}

func TestServiceCreateSuccess(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)

	expiresAt := time.Now().UTC()

	created, err := service.Create(context.Background(), CreateParams{
		ProjectID:   1,
		Name:        "test",
		ExpiresAt:   &expiresAt,
		ActorUserID: 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !store.projectCalled {
		t.Fatal("project ownership check was not called")
	}

	if store.projectArg.ProjectID != 1 {
		t.Fatalf("want checked project_id 1, got %d", store.projectArg.ProjectID)
	}

	if store.projectArg.UserID != 10 {
		t.Fatalf("want checked user_id 10, got %d", store.projectArg.UserID)
	}

	if !store.called {
		t.Fatal("store was not called")
	}

	if created.ID != 1 {
		t.Fatalf("want id 1, got %d", created.ID)
	}

	if created.ProjectID != 1 {
		t.Fatalf("want project_id 1, got %d", created.ProjectID)
	}

	if created.Name != "test" {
		t.Fatalf("want name test, got %q", created.Name)
	}

	if created.Plaintext == "" {
		t.Fatal("want plaintext to be returned")
	}

	if created.Prefix == "" {
		t.Fatal("want prefix to be returned")
	}

	if store.arg.ProjectID != 1 {
		t.Fatalf("want stored project_id 1, got %d", store.arg.ProjectID)
	}

	if store.arg.Name != "test" {
		t.Fatalf("want stored name test, got %q", store.arg.Name)
	}

	if store.arg.KeyHash == "" {
		t.Fatal("want key hash to be stored")
	}

	if store.arg.KeyHash == created.Plaintext {
		t.Fatal("stored key hash must not equal plaintext")
	}

	if store.arg.KeyPrefix != created.Prefix {
		t.Fatalf("want stored prefix %q, got %q", created.Prefix, store.arg.KeyPrefix)
	}

	if !store.arg.ExpiresAt.Valid {
		t.Fatal("want expires_at to be valid")
	}

	if created.ExpiresAt == nil {
		t.Fatal("want returned expires_at")
	}
}

func TestServiceCreateInvalidProjectID(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)

	created, err := service.Create(context.Background(), CreateParams{
		ProjectID:   -1,
		Name:        "test",
		ActorUserID: 10,
	})

	if created != nil {
		t.Fatal("want created key to be nil")
	}

	if !errors.Is(err, ErrInvalidProjectID) {
		t.Fatalf("want ErrInvalidProjectID, got %v", err)
	}

	if store.projectCalled {
		t.Fatal("want project ownership check not to be called")
	}

	if store.called {
		t.Fatal("want store not to be called")
	}
}

func TestServiceCreateInvalidActorUserID(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)

	created, err := service.Create(context.Background(), CreateParams{
		ProjectID:   1,
		Name:        "test",
		ActorUserID: 0,
	})

	if created != nil {
		t.Fatal("want created key to be nil")
	}

	if !errors.Is(err, ErrUnauthorizedProject) {
		t.Fatalf("want ErrUnauthorizedProject, got %v", err)
	}

	if store.projectCalled {
		t.Fatal("want project ownership check not to be called")
	}

	if store.called {
		t.Fatal("want store not to be called")
	}
}

func TestServiceCreateUnauthorizedProject(t *testing.T) {
	store := &fakeStore{
		projectErr: pgx.ErrNoRows,
	}
	service := NewService(store)

	created, err := service.Create(context.Background(), CreateParams{
		ProjectID:   1,
		Name:        "test",
		ActorUserID: 10,
	})

	if created != nil {
		t.Fatal("want created key to be nil")
	}

	if !errors.Is(err, ErrUnauthorizedProject) {
		t.Fatalf("want ErrUnauthorizedProject, got %v", err)
	}

	if !store.projectCalled {
		t.Fatal("want project ownership check to be called")
	}

	if store.called {
		t.Fatal("want store not to be called")
	}
}

func TestServiceCreateProjectCheckStoreError(t *testing.T) {
	storeErr := errors.New("select project failed")
	store := &fakeStore{
		projectErr: storeErr,
	}
	service := NewService(store)

	created, err := service.Create(context.Background(), CreateParams{
		ProjectID:   1,
		Name:        "test",
		ActorUserID: 10,
	})

	if created != nil {
		t.Fatal("want created key to be nil")
	}

	if !errors.Is(err, storeErr) {
		t.Fatalf("want store error, got %v", err)
	}

	if !store.projectCalled {
		t.Fatal("want project ownership check to be called")
	}

	if store.called {
		t.Fatal("want store not to be called")
	}
}

func TestServiceCreateInvalidName(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)

	created, err := service.Create(context.Background(), CreateParams{
		ProjectID:   1,
		Name:        "   ",
		ActorUserID: 10,
	})

	if created != nil {
		t.Fatal("want created key to be nil")
	}

	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("want ErrInvalidName, got %v", err)
	}

	if !store.projectCalled {
		t.Fatal("want project ownership check to be called")
	}

	if store.called {
		t.Fatal("want store not to be called")
	}

}

func TestServiceCreateStoreError(t *testing.T) {
	storeErr := errors.New("insert api key failed")
	store := &fakeStore{err: storeErr}
	service := NewService(store)

	created, err := service.Create(context.Background(), CreateParams{
		ProjectID:   1,
		Name:        "test",
		ActorUserID: 10,
	})

	if created != nil {
		t.Fatal("want created key to be nil")
	}

	if !errors.Is(err, storeErr) {
		t.Fatalf("want store error, got %v", err)
	}

	if !store.called {
		t.Fatal("want store to be called")
	}
}

func TestServiceCreateWithoutExpiresAt(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)

	created, err := service.Create(context.Background(), CreateParams{
		ProjectID:   1,
		Name:        "test",
		ExpiresAt:   nil,
		ActorUserID: 10,
	})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}

	if store.arg.ExpiresAt.Valid {
		t.Fatal("want stored expires_at to be null")
	}

	if created.ExpiresAt != nil {
		t.Fatal("want returned expires_at to be nil")
	}
}
