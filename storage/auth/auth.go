package auth

type AuthResult struct {
	UserId    int64
	UserLogin string
	ExpiresAt int64 // in GMT. in seconds for centrifuge
}
