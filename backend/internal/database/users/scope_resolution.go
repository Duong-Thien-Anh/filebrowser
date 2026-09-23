package users

import (
	"fmt"
	"path"
	"strings"
)

// ResolvedScope describes the scope that applies to one requested path.
// IndexPath is always the absolute path within the configured source. DisplayScope
// is the prefix that should be stripped from a response path before it is sent to
// the frontend.
type ResolvedScope struct {
	IndexPath       string
	DisplayScope    string
	Permissions     SourceFilePermissions
	Virtual         bool
	CandidateScopes []string
}

// ResolveScopeForSourcePath resolves a request against all scopes assigned to a
// source. A request inside a configured scope uses the most specific scope. A
// request at a common parent of multiple scopes is treated as a read-only virtual
// parent, so navigating to that parent never prepends the first child scope.
func (u *User) ResolveScopeForSourcePath(sourcePath, requestPath string) (ResolvedScope, error) {
	if u == nil {
		return ResolvedScope{}, fmt.Errorf("user not provided")
	}

	requestPath = normalizeScope(requestPath)
	var sourceScopes []BackendScope
	for _, scope := range u.BackendScopes {
		if scope.Path == sourcePath {
			scope.Scope = normalizeScope(scope.Scope)
			sourceScopes = append(sourceScopes, scope)
		}
	}
	if len(sourceScopes) == 0 {
		return ResolvedScope{}, fmt.Errorf("scope not found for source %v", sourcePath)
	}

	// Prefer the longest matching configured scope. This also handles nested
	// scopes without relying on their persistence order.
	var best *BackendScope
	for i := range sourceScopes {
		scope := &sourceScopes[i]
		if !pathContains(scope.Scope, requestPath) {
			continue
		}
		if best == nil || len(scope.Scope) > len(best.Scope) {
			best = scope
		}
	}
	if best != nil {
		return ResolvedScope{
			IndexPath:       requestPath,
			DisplayScope:    best.Scope,
			Permissions:     effectiveScopePermissions(*best),
			CandidateScopes: []string{best.Scope},
		}, nil
	}

	// A path above one or more assigned scopes is navigable, but it must not
	// inherit write permissions from a child scope. Keep only permissions that
	// are safe for the shared parent; create/modify/delete remain disabled.
	var descendants []BackendScope
	for _, scope := range sourceScopes {
		if pathContains(requestPath, scope.Scope) {
			descendants = append(descendants, scope)
		}
	}
	if len(descendants) > 0 {
		permissions := readOnlyIntersection(descendants)
		candidateScopes := make([]string, 0, len(descendants))
		for _, scope := range descendants {
			candidateScopes = append(candidateScopes, scope.Scope)
		}
		return ResolvedScope{
			IndexPath: requestPath,
			// Keep virtual-parent listings absolute. Stripping the parent here
			// would make a child such as /Marketing ambiguous on the next API
			// request when multiple department scopes share this source.
			DisplayScope:    "/",
			Permissions:     permissions,
			Virtual:         true,
			CandidateScopes: candidateScopes,
		}, nil
	}

	// Preserve the legacy relative-path contract for a single scoped source.
	// For multiple scopes an ambiguous path is rejected rather than being
	// silently attached to the first department.
	if len(sourceScopes) == 1 {
		scope := sourceScopes[0]
		return ResolvedScope{
			IndexPath:       joinScope(scope.Scope, requestPath),
			DisplayScope:    scope.Scope,
			Permissions:     effectiveScopePermissions(scope),
			CandidateScopes: []string{scope.Scope},
		}, nil
	}

	return ResolvedScope{}, fmt.Errorf("path %q is outside the user's scopes for source %v", requestPath, sourcePath)
}

// ScopeContainsOrHasChild reports whether a path is inside an assigned scope
// or is an ancestor of one. It is used to hide sibling folders at virtual
// parents while retaining normal listings inside a concrete scope.
func (u *User) ScopeContainsOrHasChild(sourcePath, requestPath string) bool {
	requestPath = normalizeScope(requestPath)
	for _, scope := range u.BackendScopes {
		if scope.Path != sourcePath {
			continue
		}
		scopePath := normalizeScope(scope.Scope)
		if pathContains(scopePath, requestPath) || pathContains(requestPath, scopePath) {
			return true
		}
	}
	return false
}

func effectiveScopePermissions(scope BackendScope) SourceFilePermissions {
	perms := scope.Permissions
	if perms.IsUnset() && scope.Permissions.Configured == false {
		return SourceFilePermissions{View: true, Download: true}
	}
	return perms
}

func readOnlyIntersection(scopes []BackendScope) SourceFilePermissions {
	permissions := SourceFilePermissions{View: true, Download: true, Configured: true}
	for _, scope := range scopes {
		perms := effectiveScopePermissions(scope)
		permissions.View = permissions.View && perms.View
		permissions.Download = permissions.Download && perms.Download
	}
	return permissions
}

func pathContains(base, candidate string) bool {
	base = normalizeScope(base)
	candidate = normalizeScope(candidate)
	return base == "/" || candidate == base || strings.HasPrefix(candidate, base+"/")
}

func joinScope(scope, requestPath string) string {
	if scope == "/" {
		return requestPath
	}
	if requestPath == "/" {
		return scope
	}
	return normalizeScope(path.Join(scope, requestPath))
}
