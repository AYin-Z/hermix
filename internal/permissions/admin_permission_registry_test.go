package permissions

import "testing"

func TestAdminPermissionRegistryMatchesMethodAndPath(t *testing.T) {
	code, ok := GetAdminPermissionCode("POST", "/api/admin/topic/list")
	if !ok {
		t.Fatalf("expected topic list permission to be registered")
	}
	if code != PermissionTopicView.Code {
		t.Fatalf("expected %s, got %s", PermissionTopicView.Code, code)
	}
}

func TestAdminPermissionRegistryAllowsRoleOptionsFromUserUpdate(t *testing.T) {
	codes, ok := GetAdminPermissionCodes("GET", "/api/admin/role/roles")
	if !ok {
		t.Fatalf("expected role options permission to be registered")
	}
	expected := []string{PermissionRoleView.Code, PermissionUserUpdate.Code}
	if len(codes) != len(expected) {
		t.Fatalf("expected %#v, got %#v", expected, codes)
	}
	for i, expectedCode := range expected {
		if codes[i] != expectedCode {
			t.Fatalf("expected %#v, got %#v", expected, codes)
		}
	}
}

func TestAdminPermissionRegistryAllowsEitherUserForbiddenPermission(t *testing.T) {
	codes, ok := GetAdminPermissionCodes("POST", "/api/admin/user/forbidden")
	if !ok {
		t.Fatalf("expected user forbidden permission to be registered")
	}
	expected := []string{PermissionUserForbidden.Code, PermissionUserForbiddenForever.Code}
	if len(codes) != len(expected) {
		t.Fatalf("expected %#v, got %#v", expected, codes)
	}
	for i, expectedCode := range expected {
		if codes[i] != expectedCode {
			t.Fatalf("expected %#v, got %#v", expected, codes)
		}
	}
}

func TestAdminPermissionRegistryProtectsForbiddenWordDelete(t *testing.T) {
	code, ok := GetAdminPermissionCode("POST", "/api/admin/forbidden-word/delete")
	if !ok {
		t.Fatalf("expected forbidden word delete permission to be registered")
	}
	if code != PermissionForbiddenWordDelete.Code {
		t.Fatalf("expected %s, got %s", PermissionForbiddenWordDelete.Code, code)
	}
}

func TestAdminPermissionRegistryProtectsBadgeUpdateSort(t *testing.T) {
	code, ok := GetAdminPermissionCode("POST", "/api/admin/badge/update_sort")
	if !ok {
		t.Fatalf("expected badge sort permission to be registered")
	}
	if code != PermissionBadgeUpdate.Code {
		t.Fatalf("expected %s, got %s", PermissionBadgeUpdate.Code, code)
	}
}

func TestAdminPermissionRegistryProtectsTaskConfigUpdateSort(t *testing.T) {
	code, ok := GetAdminPermissionCode("POST", "/api/admin/task-config/update_sort")
	if !ok {
		t.Fatalf("expected task config sort permission to be registered")
	}
	if code != PermissionTaskUpdate.Code {
		t.Fatalf("expected %s, got %s", PermissionTaskUpdate.Code, code)
	}
}

func TestAdminPermissionRegistryProtectsLinkDelete(t *testing.T) {
	code, ok := GetAdminPermissionCode("POST", "/api/admin/link/delete")
	if !ok {
		t.Fatalf("expected link delete permission to be registered")
	}
	if code != PermissionLinkDelete.Code {
		t.Fatalf("expected %s, got %s", PermissionLinkDelete.Code, code)
	}
}

func TestAdminPermissionRegistryProtectsLinkUpdateSort(t *testing.T) {
	code, ok := GetAdminPermissionCode("POST", "/api/admin/link/update_sort")
	if !ok {
		t.Fatalf("expected link sort permission to be registered")
	}
	if code != PermissionLinkUpdate.Code {
		t.Fatalf("expected %s, got %s", PermissionLinkUpdate.Code, code)
	}
}

func TestAdminPermissionRegistryProtectsUserReportAudit(t *testing.T) {
	code, ok := GetAdminPermissionCode("POST", "/api/admin/user-report/audit")
	if !ok {
		t.Fatalf("expected user report audit permission to be registered")
	}
	if code != PermissionUserReportAudit.Code {
		t.Fatalf("expected %s, got %s", PermissionUserReportAudit.Code, code)
	}
}

func TestAdminPermissionRegistryRejectsUnknownAdminPath(t *testing.T) {
	if code, ok := GetAdminPermissionCode("POST", "/api/admin/unknown/action"); ok {
		t.Fatalf("expected unknown admin path to be rejected, got %s", code)
	}
}

func TestAdminPermissionRegistryProtectsCommentManagementPaths(t *testing.T) {
	paths := []struct {
		method   string
		path     string
		expected string
	}{
		{method: "GET", path: "/api/admin/comment/1", expected: PermissionCommentView.Code},
		{method: "POST", path: "/api/admin/comment/list", expected: PermissionCommentView.Code},
		{method: "POST", path: "/api/admin/comment/audit", expected: PermissionCommentAudit.Code},
		{method: "POST", path: "/api/admin/comment/delete", expected: PermissionCommentDelete.Code},
	}

	for _, path := range paths {
		code, ok := GetAdminPermissionCode(path.method, path.path)
		if !ok {
			t.Fatalf("expected %s %s to be registered", path.method, path.path)
		}
		if code != path.expected {
			t.Fatalf("expected %s for %s %s, got %s", path.expected, path.method, path.path, code)
		}
	}

	// 未注册的方法仍然要落到拒绝分支，别被 /api/admin/comment/* 通配规则顺带放行。
	if code, ok := GetAdminPermissionCode("DELETE", "/api/admin/comment/1"); ok {
		t.Fatalf("expected DELETE /api/admin/comment/1 to be rejected, got %s", code)
	}
}
