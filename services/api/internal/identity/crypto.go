package identity

import "minIDM/internal/password"

func VerifyPassword(pw, encoded string) (bool, error) { return password.Verify(pw, encoded) }
func HashPassword(pw string) (string, error)          { return password.Hash(pw) }
