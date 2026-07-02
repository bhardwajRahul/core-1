package internal

import (
	"testing"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/model"
)

func TestReadPermissions(t *testing.T) {
	tables := make(map[string]PermissionLevel)
	tables["normal"] = PermGroup
	tables["only-owner-read_700_"] = PermOwner
	tables["same-acct-users_770_"] = PermGroup
	tables["logged-in_774_"] = PermEveryone

	for col, perm := range tables {
		if p := ReadPermission(col); p != perm {
			t.Errorf("%s: expected read perm %d got %d", col, perm, p)
		}
	}
}

func TestWritePermissions(t *testing.T) {
	tables := make(map[string]PermissionLevel)
	tables["normal"] = PermOwner
	tables["same-acct-users_770_"] = PermGroup
	tables["only-owner-write_700_"] = PermOwner
	tables["logged-in_772_"] = PermEveryone

	for col, perm := range tables {
		if p := WritePermission(col); p != perm {
			t.Errorf("%s: expected write perm %d got %d", col, perm, p)
		}
	}
}

func TestReadScopeRoleAwarePermissionsDisabled(t *testing.T) {
	withRoleAwarePermissions(t, false)

	if p := ReadScope(model.Auth{Role: 0}, "normal"); p != RowScopeAccount {
		t.Errorf("expected default read scope to be account got %d", p)
	}
	if p := WriteScope(model.Auth{Role: 50}, "only-owner-write_700_", true); p != RowScopeOwner {
		t.Errorf("expected role 50 write scope to follow octal permissions got %d", p)
	}
}

func TestReadScopeRoleAwarePermissions(t *testing.T) {
	withRoleAwarePermissions(t, true)

	tests := map[string]struct {
		auth model.Auth
		col  string
		want RowPermissionScope
	}{
		"role 0 default read is owner": {
			auth: model.Auth{Role: 0},
			col:  "normal",
			want: RowScopeOwner,
		},
		"role 0 everyone read is owner": {
			auth: model.Auth{Role: 0},
			col:  "logged-in_774_",
			want: RowScopeOwner,
		},
		"role 10 follows octal": {
			auth: model.Auth{Role: 10},
			col:  "normal",
			want: RowScopeAccount,
		},
		"role 50 owner read is account": {
			auth: model.Auth{Role: 50},
			col:  "only-owner-read_700_",
			want: RowScopeAccount,
		},
		"root is everyone": {
			auth: model.Auth{Role: 100},
			col:  "only-owner-read_700_",
			want: RowScopeEveryone,
		},
		"public is everyone": {
			auth: model.Auth{Role: 0},
			col:  "pub_tasks",
			want: RowScopeEveryone,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := ReadScope(tt.auth, tt.col); got != tt.want {
				t.Fatalf("expected %d got %d", tt.want, got)
			}
		})
	}
}

func TestWriteScopeRoleAwarePermissions(t *testing.T) {
	withRoleAwarePermissions(t, true)

	tests := map[string]struct {
		auth model.Auth
		col  string
		want RowPermissionScope
	}{
		"role 0 group write is owner": {
			auth: model.Auth{Role: 0},
			col:  "same-acct-users_770_",
			want: RowScopeOwner,
		},
		"role 10 follows octal": {
			auth: model.Auth{Role: 10},
			col:  "same-acct-users_770_",
			want: RowScopeAccount,
		},
		"role 50 owner write is account": {
			auth: model.Auth{Role: 50},
			col:  "only-owner-write_700_",
			want: RowScopeAccount,
		},
		"root is everyone": {
			auth: model.Auth{Role: 100},
			col:  "only-owner-write_700_",
			want: RowScopeEveryone,
		},
		"public write follows provider setting": {
			auth: model.Auth{Role: 0},
			col:  "pub_tasks",
			want: RowScopeEveryone,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := WriteScope(tt.auth, tt.col, true); got != tt.want {
				t.Fatalf("expected %d got %d", tt.want, got)
			}
		})
	}
}

func withRoleAwarePermissions(t *testing.T, enabled bool) {
	t.Helper()

	orig := config.Current.RoleAwareRowPermissions
	config.Current.RoleAwareRowPermissions = enabled
	t.Cleanup(func() {
		config.Current.RoleAwareRowPermissions = orig
	})
}
