package auth

import "testing"

func TestHashCheckPasswordHash_RoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPasswordHash("correct horse battery staple", hash) {
		t.Error("CheckPasswordHash with correct password: want true, got false")
	}
}

func TestCheckPasswordHash_WrongPasswordFails(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if CheckPasswordHash("wrong password", hash) {
		t.Error("CheckPasswordHash with wrong password: want false, got true")
	}
}

func TestHashPassword_SaltIsRandom(t *testing.T) {
	h1, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	h2, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if h1 == h2 {
		t.Error("two hashes of the same password are identical, want different salts")
	}
	// Both must still verify independently.
	if !CheckPasswordHash("same-password", h1) || !CheckPasswordHash("same-password", h2) {
		t.Error("one of the two independently-salted hashes failed to verify")
	}
}

func TestCheckPasswordHash_MalformedInputsReturnFalse(t *testing.T) {
	cases := map[string]string{
		"empty string":         "",
		"not delimited at all": "not-a-valid-hash",
		"wrong scheme":         "$bcrypt$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA",
		"too few segments":     "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA",
		"bad param format":     "$argon2id$v=19$m65536t3p4$c2FsdA$aGFzaA",
		"unknown param key":    "$argon2id$v=19$m=65536,t=3,x=4$c2FsdA$aGFzaA",
		"zero param":           "$argon2id$v=19$m=0,t=3,p=4$c2FsdA$aGFzaA",
		"threads over max":     "$argon2id$v=19$m=65536,t=3,p=999$c2FsdA$aGFzaA",
		"bad salt base64":      "$argon2id$v=19$m=65536,t=3,p=4$not-b64!!$aGFzaA",
		"bad hash base64":      "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$not-b64!!",
	}
	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("CheckPasswordHash panicked on %q input: %v", name, r)
				}
			}()
			if CheckPasswordHash("whatever", encoded) {
				t.Errorf("CheckPasswordHash(%q) = true, want false", encoded)
			}
		})
	}
}

func TestHashPassword_EmptyPasswordRoundTrips(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword(\"\"): %v", err)
	}
	if !CheckPasswordHash("", hash) {
		t.Error("CheckPasswordHash(\"\", hash of \"\"): want true, got false")
	}
	if CheckPasswordHash("not-empty", hash) {
		t.Error("CheckPasswordHash(\"not-empty\", hash of \"\"): want false, got true")
	}
}
