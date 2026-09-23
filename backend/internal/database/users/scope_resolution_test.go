package users

import "testing"

func TestResolveScopeForSourcePathUsesMostSpecificScope(t *testing.T) {
	user := &User{BackendScopes: []BackendScope{
		{Path: "/data/filebrowser", Scope: "/Sài Gòn An Thái/Marketing", Permissions: SourceFilePermissions{View: true, Download: true, Modify: true, Create: true, Configured: true}},
		{Path: "/data/filebrowser", Scope: "/Sài Gòn An Thái/SGAT - Data Chung", Permissions: SourceFilePermissions{View: true, Download: true, Modify: true, Create: true, Configured: true}},
	}}

	resolved, err := user.ResolveScopeForSourcePath("/data/filebrowser", "/Sài Gòn An Thái/Marketing/brief.docx")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.IndexPath != "/Sài Gòn An Thái/Marketing/brief.docx" {
		t.Fatalf("resolved index path = %q", resolved.IndexPath)
	}
	if resolved.DisplayScope != "/Sài Gòn An Thái/Marketing" {
		t.Fatalf("display scope = %q", resolved.DisplayScope)
	}
	if resolved.Virtual {
		t.Fatal("concrete child path must not be virtual")
	}
	if !resolved.Permissions.Create || !resolved.Permissions.Modify {
		t.Fatalf("child permissions = %+v", resolved.Permissions)
	}
}

func TestResolveScopeForSourcePathUsesCommonParentWithoutPrefixingFirstChild(t *testing.T) {
	user := &User{BackendScopes: []BackendScope{
		{Path: "/data/filebrowser", Scope: "/Sài Gòn An Thái/Marketing", Permissions: SourceFilePermissions{View: true, Download: true, Modify: true, Create: true, Configured: true}},
		{Path: "/data/filebrowser", Scope: "/Sài Gòn An Thái/SGAT - Data Chung", Permissions: SourceFilePermissions{View: true, Download: true, Modify: true, Create: true, Configured: true}},
	}}

	resolved, err := user.ResolveScopeForSourcePath("/data/filebrowser", "/Sài Gòn An Thái")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.IndexPath != "/Sài Gòn An Thái" {
		t.Fatalf("resolved parent path = %q", resolved.IndexPath)
	}
	if resolved.DisplayScope != "/" || !resolved.Virtual {
		t.Fatalf("resolved parent = %+v", resolved)
	}
	if resolved.Permissions.Modify || resolved.Permissions.Create || resolved.Permissions.Delete {
		t.Fatalf("parent must be read-only, got %+v", resolved.Permissions)
	}
	if !resolved.Permissions.View || !resolved.Permissions.Download {
		t.Fatalf("parent should retain view/download, got %+v", resolved.Permissions)
	}
}

func TestResolveScopeForSourcePathRejectsAmbiguousMultiScopePath(t *testing.T) {
	user := &User{BackendScopes: []BackendScope{
		{Path: "/data/filebrowser", Scope: "/Root/Marketing", Permissions: SourceFilePermissions{View: true, Configured: true}},
		{Path: "/data/filebrowser", Scope: "/Root/Finance", Permissions: SourceFilePermissions{View: true, Configured: true}},
	}}

	if _, err := user.ResolveScopeForSourcePath("/data/filebrowser", "/Root/Unknown"); err == nil {
		t.Fatal("expected an ambiguous path outside all assigned scopes to be rejected")
	}
}

func TestScopeContainsOrHasChildHidesUnassignedSibling(t *testing.T) {
	user := &User{BackendScopes: []BackendScope{
		{Path: "/data/filebrowser", Scope: "/Root/Marketing"},
	}}

	if !user.ScopeContainsOrHasChild("/data/filebrowser", "/Root") {
		t.Fatal("expected assigned scope to be visible through its parent")
	}
	if user.ScopeContainsOrHasChild("/data/filebrowser", "/Root/Finance") {
		t.Fatal("unassigned sibling must not be visible")
	}
}

func TestScopeHasDescendantKeepsIntermediateFoldersVisible(t *testing.T) {
	user := &User{BackendScopes: []BackendScope{
		{Path: "/data/filebrowser", Scope: "/Root/Team/Marketing"},
	}}

	if !user.ScopeHasDescendant("/data/filebrowser", "/Root") {
		t.Fatal("expected root ancestor to lead to an assigned scope")
	}
	if !user.ScopeHasDescendant("/data/filebrowser", "/Root/Team") {
		t.Fatal("expected intermediate ancestor to lead to an assigned scope")
	}
	if user.ScopeHasDescendant("/data/filebrowser", "/Root/Team/Marketing") {
		t.Fatal("assigned scope itself is not a descendant")
	}
	if user.ScopeHasDescendant("/data/filebrowser", "/Root/Team/Finance") {
		t.Fatal("sibling path must not be treated as an ancestor")
	}
}
