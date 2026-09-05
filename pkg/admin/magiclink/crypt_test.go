package magiclink

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	in := struct {
		UUID string
		Name string
	}{UUID: "abc", Name: "guest"}

	token, err := New(key).Encrypt(&in)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty token")
	}

	var out struct {
		UUID string
		Name string
	}
	if err := New(key).Decrypt(token, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("got %+v want %+v", out, in)
	}
}
