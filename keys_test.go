package godo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestKeys_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/account/keys", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"ssh_keys":[{"id":1},{"id":2}]}   `)
	})

	keys, _, err := client.Keys.List()
	if err != nil {
		t.Errorf("Keys.List returned error: %v", err)
	}

	expected := []Key{{ID: 1}, {ID: 2}}
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Keys.List returned %+v, expected %+v", keys, expected)
	}
}

func TestKeys_GetByID(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/account/keys/12345", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"ssh_key": {"id":12345}}`)
	})

	keys, _, err := client.Keys.GetByID(12345)
	if err != nil {
		t.Errorf("Keys.GetByID returned error: %v", err)
	}

	expected := &Key{ID: 12345}
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Keys.GetByID returned %+v, expected %+v", keys, expected)
	}
}

func TestKeys_GetByFingerprint(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/account/keys/aa:bb:cc", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"ssh_key": {"fingerprint":"aa:bb:cc"}}`)
	})

	keys, _, err := client.Keys.GetByFingerprint("aa:bb:cc")
	if err != nil {
		t.Errorf("Keys.GetByFingerprint returned error: %v", err)
	}

	expected := &Key{Fingerprint: "aa:bb:cc"}
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Keys.GetByFingerprint returned %+v, expected %+v", keys, expected)
	}
}

func TestKeys_Create(t *testing.T) {
	setup()
	defer teardown()

	createRequest := &KeyCreateRequest{
		Name:      "name",
		PublicKey: "ssh-rsa longtextandstuff",
	}

	mux.HandleFunc("/v2/account/keys", func(w http.ResponseWriter, r *http.Request) {
		v := new(KeyCreateRequest)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "POST")
		if !reflect.DeepEqual(v, createRequest) {
			t.Errorf("Request body = %+v, expected %+v", v, createRequest)
		}

		fmt.Fprintf(w, `{"ssh_key":{"id":1}}`)
	})

	key, _, err := client.Keys.Create(createRequest)
	if err != nil {
		t.Errorf("Keys.Create returned error: %v", err)
	}

	expected := &Key{ID: 1}
	if !reflect.DeepEqual(key, expected) {
		t.Errorf("Keys.Create returned %+v, expected %+v", key, expected)
	}
}

func TestKeys_DestroyByID(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/account/keys/12345", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
	})

	_, err := client.Keys.DeleteByID(12345)
	if err != nil {
		t.Errorf("Keys.Delete returned error: %v", err)
	}
}

func TestKeys_DestroyByFingerprint(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/account/keys/aa:bb:cc", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
	})

	_, err := client.Keys.DeleteByFingerprint("aa:bb:cc")
	if err != nil {
		t.Errorf("Keys.Delete returned error: %v", err)
	}
}

func TestKey_String(t *testing.T) {
	key := &Key{
		ID:          123,
		Name:        "Key",
		Fingerprint: "fingerprint",
		PublicKey:   "public key",
	}

	stringified := key.String()
	expected := `godo.Key{ID:123, Name:"Key", Fingerprint:"fingerprint", PublicKey:"public key"}`
	if expected != stringified {
		t.Errorf("Key.String returned %+v, expected %+v", stringified, expected)
	}
}
