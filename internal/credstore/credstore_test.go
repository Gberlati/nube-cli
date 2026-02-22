package credstore

import (
	"os"
	"testing"
)

func setupTempDir(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
}

func TestWriteRead_Roundtrip(t *testing.T) {
	setupTempDir(t)

	f := File{
		DefaultStore: "my-shop",
		Stores: map[string]StoreProfile{
			"my-shop": {
				StoreID:     "1234",
				AccessToken: "tok-abc",
				Email:       "owner@shop.com",
				Scopes:      []string{"read_products"},
				CreatedAt:   "2025-01-01T00:00:00Z",
			},
		},
		OAuthClients: map[string]OAuthClient{
			"default": {ClientID: "cid", ClientSecret: "csec"},
		},
	}

	if err := Write(f); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.DefaultStore != "my-shop" {
		t.Errorf("DefaultStore = %q, want %q", got.DefaultStore, "my-shop")
	}

	if got.Stores["my-shop"].AccessToken != "tok-abc" {
		t.Errorf("AccessToken = %q", got.Stores["my-shop"].AccessToken)
	}

	if got.OAuthClients["default"].ClientID != "cid" {
		t.Errorf("ClientID = %q", got.OAuthClients["default"].ClientID)
	}
}

func TestRead_NoFile(t *testing.T) {
	setupTempDir(t)

	f, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if len(f.Stores) != 0 {
		t.Errorf("expected empty stores, got %d", len(f.Stores))
	}
}

func TestWrite_Permissions(t *testing.T) {
	setupTempDir(t)

	if err := Write(File{}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	path, _ := Path()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %o, want 0600", perm)
	}
}

func TestSetStore_AutoDefault(t *testing.T) {
	setupTempDir(t)

	err := SetStore("first", StoreProfile{StoreID: "1", AccessToken: "t1"})
	if err != nil {
		t.Fatalf("SetStore: %v", err)
	}

	f, _ := Read()
	if f.DefaultStore != "first" {
		t.Errorf("DefaultStore = %q, want %q", f.DefaultStore, "first")
	}

	// Second store should not change default.
	err = SetStore("second", StoreProfile{StoreID: "2", AccessToken: "t2"})
	if err != nil {
		t.Fatalf("SetStore second: %v", err)
	}

	f, _ = Read()
	if f.DefaultStore != "first" {
		t.Errorf("DefaultStore = %q, want %q after second store", f.DefaultStore, "first")
	}
}

func TestGetStore_NotFound(t *testing.T) {
	setupTempDir(t)

	_, err := GetStore("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRemoveStore(t *testing.T) {
	setupTempDir(t)

	_ = SetStore("s1", StoreProfile{StoreID: "1", AccessToken: "t"})
	_ = SetStore("s2", StoreProfile{StoreID: "2", AccessToken: "t"})
	_ = SetDefault("s1")

	if err := RemoveStore("s1"); err != nil {
		t.Fatalf("RemoveStore: %v", err)
	}

	f, _ := Read()
	if _, ok := f.Stores["s1"]; ok {
		t.Error("s1 should be deleted")
	}

	// Single remaining store should auto-become default.
	if f.DefaultStore != "s2" {
		t.Errorf("DefaultStore = %q, want %q", f.DefaultStore, "s2")
	}
}

func TestRemoveStore_NotFound(t *testing.T) {
	setupTempDir(t)

	err := RemoveStore("nope")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveStore_Flag(t *testing.T) {
	setupTempDir(t)

	_ = SetStore("a", StoreProfile{StoreID: "1", AccessToken: "ta"})
	_ = SetStore("b", StoreProfile{StoreID: "2", AccessToken: "tb"})

	name, p, err := ResolveStore("b")
	if err != nil {
		t.Fatalf("ResolveStore: %v", err)
	}

	if name != "b" || p.AccessToken != "tb" {
		t.Errorf("got name=%q token=%q", name, p.AccessToken)
	}
}

func TestResolveStore_Env(t *testing.T) {
	setupTempDir(t)

	_ = SetStore("a", StoreProfile{StoreID: "1", AccessToken: "ta"})
	_ = SetStore("b", StoreProfile{StoreID: "2", AccessToken: "tb"})

	t.Setenv("NUBE_STORE", "b")

	name, p, err := ResolveStore("")
	if err != nil {
		t.Fatalf("ResolveStore: %v", err)
	}

	if name != "b" || p.AccessToken != "tb" {
		t.Errorf("got name=%q token=%q", name, p.AccessToken)
	}
}

func TestResolveStore_Default(t *testing.T) {
	setupTempDir(t)

	_ = SetStore("a", StoreProfile{StoreID: "1", AccessToken: "ta"})
	_ = SetStore("b", StoreProfile{StoreID: "2", AccessToken: "tb"})
	_ = SetDefault("a")

	name, p, err := ResolveStore("")
	if err != nil {
		t.Fatalf("ResolveStore: %v", err)
	}

	if name != "a" || p.AccessToken != "ta" {
		t.Errorf("got name=%q token=%q", name, p.AccessToken)
	}
}

func TestResolveStore_SingleAutoSelect(t *testing.T) {
	setupTempDir(t)

	_ = SetStore("only", StoreProfile{StoreID: "1", AccessToken: "tok"})

	name, p, err := ResolveStore("")
	if err != nil {
		t.Fatalf("ResolveStore: %v", err)
	}

	if name != "only" || p.AccessToken != "tok" {
		t.Errorf("got name=%q token=%q", name, p.AccessToken)
	}
}

func TestResolveStore_Empty(t *testing.T) {
	setupTempDir(t)

	_, _, err := ResolveStore("")
	if err == nil {
		t.Fatal("expected error for empty store")
	}
}

func TestResolveStore_Ambiguous(t *testing.T) {
	setupTempDir(t)

	_ = Write(File{
		Stores: map[string]StoreProfile{
			"a": {StoreID: "1", AccessToken: "ta"},
			"b": {StoreID: "2", AccessToken: "tb"},
		},
	})

	_, _, err := ResolveStore("")
	if err == nil {
		t.Fatal("expected error for ambiguous store")
	}
}

func TestListStores(t *testing.T) {
	setupTempDir(t)

	_ = SetStore("beta", StoreProfile{StoreID: "2", AccessToken: "t"})
	_ = SetStore("alpha", StoreProfile{StoreID: "1", AccessToken: "t"})

	names, err := ListStores()
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}

	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("got %v", names)
	}
}

func TestSetDefault(t *testing.T) {
	setupTempDir(t)

	_ = SetStore("a", StoreProfile{StoreID: "1", AccessToken: "t"})
	_ = SetStore("b", StoreProfile{StoreID: "2", AccessToken: "t"})

	if err := SetDefault("b"); err != nil {
		t.Fatalf("SetDefault: %v", err)
	}

	f, _ := Read()
	if f.DefaultStore != "b" {
		t.Errorf("DefaultStore = %q, want %q", f.DefaultStore, "b")
	}
}

func TestSetDefault_NotFound(t *testing.T) {
	setupTempDir(t)

	err := SetDefault("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOAuthClient_SetGet(t *testing.T) {
	setupTempDir(t)

	err := SetOAuthClient("default", OAuthClient{ClientID: "cid", ClientSecret: "csec"})
	if err != nil {
		t.Fatalf("SetOAuthClient: %v", err)
	}

	c, err := GetOAuthClient("default")
	if err != nil {
		t.Fatalf("GetOAuthClient: %v", err)
	}

	if c.ClientID != "cid" || c.ClientSecret != "csec" {
		t.Errorf("got %+v", c)
	}
}

func TestGetOAuthClient_Missing(t *testing.T) {
	setupTempDir(t)

	_, err := GetOAuthClient("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}

	var missing *OAuthClientMissingError
	if !isOAuthClientMissing(err, &missing) {
		t.Errorf("expected OAuthClientMissingError, got %T", err)
	}
}

func isOAuthClientMissing(err error, target **OAuthClientMissingError) bool {
	e, ok := err.(*OAuthClientMissingError)
	if ok {
		*target = e
	}

	return ok
}
