package internal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/model"
)

type PermissionLevel int

const (
	PermOwner PermissionLevel = iota
	PermGroup
	PermEveryone
)

type RowPermissionScope int

const (
	RowScopeOwner RowPermissionScope = iota
	RowScopeAccount
	RowScopeEveryone
)

func GetPermission(col string) (owner string, group string, everyone string) {
	// default permission
	owner, group, everyone = "7", "4", "0"

	re := regexp.MustCompile(`_\d\d\d_$`)
	if !re.MatchString(col) {
		return
	}

	results := re.FindAllString(col, -1)
	if len(results) != 1 {
		return
	}

	perm := strings.ReplaceAll(results[0], "_", "")

	if len(perm) != 3 {
		return
	}

	owner = string(perm[0])
	group = string(perm[1])
	everyone = string(perm[2])
	return
}

func WritePermission(col string) PermissionLevel {
	_, g, e := GetPermission(col)

	if CanWrite(e) {
		return PermEveryone
	}
	if CanWrite(g) {
		return PermGroup
	}
	return PermOwner
}

func ReadPermission(col string) PermissionLevel {
	_, g, e := GetPermission(col)

	if CanRead(e) {
		return PermEveryone
	}
	if CanRead(g) {
		return PermGroup
	}
	return PermOwner
}

func ReadScope(auth model.Auth, col string) RowPermissionScope {
	if strings.HasPrefix(col, "pub_") || auth.Role == 100 {
		return RowScopeEveryone
	}

	if config.Current.RoleAwareRowPermissions {
		if auth.Role >= 50 {
			return RowScopeAccount
		}
		if auth.Role == 0 {
			return RowScopeOwner
		}
	}

	return scopeFromPermission(ReadPermission(col))
}

func WriteScope(auth model.Auth, col string, publicWrite bool) RowPermissionScope {
	if auth.Role == 100 || (publicWrite && strings.HasPrefix(col, "pub_")) {
		return RowScopeEveryone
	}

	if config.Current.RoleAwareRowPermissions && !strings.HasPrefix(col, "pub_") {
		if auth.Role >= 50 {
			return RowScopeAccount
		}
		if auth.Role == 0 {
			return RowScopeOwner
		}
	}

	return scopeFromPermission(WritePermission(col))
}

func scopeFromPermission(perm PermissionLevel) RowPermissionScope {
	switch perm {
	case PermGroup:
		return RowScopeAccount
	case PermEveryone:
		return RowScopeEveryone
	default:
		return RowScopeOwner
	}
}

func CanWrite(s string) bool {
	i, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	return uint8(i)&uint8(2) != 0
}

func CanRead(s string) bool {
	i, err := strconv.Atoi(s)
	if err != nil {
		fmt.Println(err)
	}
	return uint8(i)&uint8(4) != 0
}
