package auth

type AuthResult struct {
	UserId      int64
	UserLogin   string
	ExpiresAt   int64 // in GMT. in seconds for centrifuge
	Roles       []string
	Permissions []string
}

func (r *AuthResult) HasRole(roleToCheck string) bool {
	var role = false
	for _, r := range r.Roles {
		if r == roleToCheck {
			role = true
			break
		}
	}
	return role
}

func (r *AuthResult) HasPermission(permissionToCheck string) bool {
	var role = false
	for _, p := range r.Permissions {
		if p == permissionToCheck {
			role = true
			break
		}
	}
	return role
}
