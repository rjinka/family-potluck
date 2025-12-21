package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"family-potluck/backend/internal/database"
	"family-potluck/backend/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/api/idtoken"
)

type MockTokenValidator struct {
	ValidateFunc func(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error)
}

func (m *MockTokenValidator) Validate(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error) {
	return m.ValidateFunc(ctx, idToken, audience)
}

func TestLogout(t *testing.T) {
	server := NewServer(nil, nil)
	req, _ := http.NewRequest("POST", "/auth/logout", nil)
	rr := httptest.NewRecorder()

	server.Logout(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "Logged out"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// Check if cookie is cleared
	cookies := rr.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "token" && c.Value == "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("token cookie not cleared")
	}
}

func TestGetMe_Unauthorized(t *testing.T) {
	server := NewServer(nil, nil)
	req, _ := http.NewRequest("GET", "/auth/me", nil)
	rr := httptest.NewRecorder()

	server.GetMe(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestGoogleLogin_NewUser(t *testing.T) {
	mockDB := &database.MockService{}
	mockValidator := &MockTokenValidator{}
	server := NewServer(mockDB, nil)
	server.TokenValidator = mockValidator

	familyID := primitive.NewObjectID()
	email := "test@example.com"

	mockValidator.ValidateFunc = func(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error) {
		return &idtoken.Payload{
			Claims: map[string]interface{}{
				"email":   email,
				"name":    "Test User",
				"picture": "http://example.com/pic.jpg",
			},
			Subject: "google-id-123",
		}, nil
	}

	mockDB.GetFamilyMemberByEmailFunc = func(ctx context.Context, e string) (*models.FamilyMember, error) {
		return nil, database.ErrNoDocuments // Assuming you have this or use mongo.ErrNoDocuments
	}

	mockDB.CreateFamilyMemberFunc = func(ctx context.Context, f *models.FamilyMember) error {
		f.ID = familyID
		return nil
	}

	loginReq := struct {
		IDToken string `json:"id_token"`
	}{IDToken: "fake-token"}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/auth/google", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	server.GoogleLogin(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var resp models.FamilyMember
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Email != email {
		t.Errorf("expected email %v, got %v", email, resp.Email)
	}
}
