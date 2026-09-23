package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gtsteffaniak/filebrowser/backend/internal/database/users"
)

func TestVerifyActorPasswordAllowsBearerAdminToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/api/users?username=digitalmkt", nil)
	req.Header.Set("Authorization", "Bearer admin-api-token")

	d := &Context{
		User: &users.User{
			FrontendUser: users.FrontendUser{
				Permissions: users.Permissions{Admin: true},
			},
		},
		Token: "admin-api-token",
	}

	status, err := verifyActorPasswordForUserActions(req, d)
	if status != 0 || err != nil {
		t.Fatalf("verifyActorPasswordForUserActions() = (%d, %v), want (0, nil)", status, err)
	}
}
