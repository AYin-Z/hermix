package services

import "testing"

func TestValidateWebhookURL_RejectsInternal(t *testing.T) {
	reject := []string{
		"http://127.0.0.1:9099/hook",
		"http://localhost/hook",
		"http://10.0.0.5/x",
		"http://192.168.1.10/x",
		"http://172.16.0.1/x",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/x",
		"http://0.0.0.0/x",
		"ftp://example.com/x",
		"not-a-url",
		"http:///nohost",
	}
	for _, u := range reject {
		if err := ValidateWebhookURL(u); err == nil {
			t.Errorf("expected reject but allowed: %s", u)
		}
	}
}

func TestSignPayload(t *testing.T) {
	body := []byte(`{"event":"comment.topic"}`)
	if got := signPayload("", body); got != "" {
		t.Errorf("empty secret should yield empty sig, got %q", got)
	}
	sig1 := signPayload("secret-a", body)
	sig2 := signPayload("secret-a", body)
	sig3 := signPayload("secret-b", body)
	if sig1 == "" || sig1 != sig2 {
		t.Errorf("HMAC not deterministic: %q vs %q", sig1, sig2)
	}
	if sig1 == sig3 {
		t.Error("different secrets must produce different signatures")
	}
}
