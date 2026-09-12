package env

import "testing"

func TestGet(t *testing.T) {
	t.Run("configured value", func(t *testing.T) {
		t.Setenv("TEST_ENV_VALUE", "configured")
		if got := Get("TEST_ENV_VALUE", "fallback"); got != "configured" {
			t.Fatalf("Get() = %q, want configured", got)
		}
	})

	t.Run("missing value", func(t *testing.T) {
		if got := Get("TEST_ENV_MISSING", "fallback"); got != "fallback" {
			t.Fatalf("Get() = %q, want fallback", got)
		}
	})

	t.Run("empty value", func(t *testing.T) {
		t.Setenv("TEST_ENV_EMPTY", "")
		if got := Get("TEST_ENV_EMPTY", "fallback"); got != "fallback" {
			t.Fatalf("Get() = %q, want fallback", got)
		}
	})
}

func TestRequired(t *testing.T) {
	t.Run("configured value", func(t *testing.T) {
		t.Setenv("TEST_REQUIRED_VALUE", "configured")
		got, err := Required("TEST_REQUIRED_VALUE")
		if err != nil || got != "configured" {
			t.Fatalf("Required() = %q, %v, want configured, nil", got, err)
		}
	})

	t.Run("missing value", func(t *testing.T) {
		if _, err := Required("TEST_REQUIRED_MISSING"); err == nil {
			t.Fatal("Required() error = nil, want error")
		}
	})

	t.Run("empty value", func(t *testing.T) {
		t.Setenv("TEST_REQUIRED_EMPTY", "")
		if _, err := Required("TEST_REQUIRED_EMPTY"); err == nil {
			t.Fatal("Required() error = nil, want error")
		}
	})
}
