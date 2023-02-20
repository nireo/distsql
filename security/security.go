package security

import (
	"encoding/json"
	"io"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// AllUser represents the permissions any user has in the auth store.
const AllUser = "*"

// Creds is used to parse credential entries from JSON.
type Creds struct {
	Username    string   `json:"username,omitempty"`
	Password    string   `json:"password,omitempty"`
	Permissions []string `json:"perms,omitempty"`
}

// CredentialsStore is used to handle different permissions of users. It adds very
// basic authentication using the authorization header in HTTP requests.
type CredentialsStore struct {
	passwords map[string][]byte

	// use struct{} since it's more efficient compared to bool
	permissions map[string]map[string]struct{}
}

// NewAuthStore returns an AuthStore pointer with the maps initialized.
func NewAuthStore() CredentialsStore {
	return CredentialsStore{
		passwords:   make(map[string][]byte),
		permissions: make(map[string]map[string]struct{}),
	}
}

// Initialize reads JSON from a given io.Reader and populates the fields
// of AuthStore in the process of reading the file.
func (cs *CredentialsStore) Initialize(r io.Reader) error {
	decoder := json.NewDecoder(r)
	if _, err := decoder.Token(); err != nil {
		return err
	}

	var cred Creds
	for decoder.More() {
		if err := decoder.Decode(&cred); err != nil {
			return err
		}

		cs.passwords[cred.Username] = []byte(cred.Password)
		cs.permissions[cred.Username] = make(map[string]struct{}, len(cred.Permissions))
		for _, perm := range cred.Permissions {
			cs.permissions[cred.Username][perm] = struct{}{}
		}
	}

	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

// SetForAllUser sets given permissions for every user in the credential store.
// This is done by using a identifier for all users, to make code faster and
// simpler.
func (cs *CredentialsStore) SetForAllUser(perms ...string) {
	cs.passwords[AllUser] = nil
	for _, perm := range perms {
		cs.permissions[AllUser][perm] = struct{}{}
	}
}

// ValidatePassword checks the passwords lists and compares the bcrypt hash of
// given password and the one in stored in the AuthStore.
func (cs *CredentialsStore) ValidatePassword(username string, password []byte) bool {
	pass, ok := cs.passwords[username]
	if !ok {
		return false
	}

	return bcrypt.CompareHashAndPassword(pass, password) == nil
}

// CheckReq reads the auth information from the request header and
// checks if the password matches.
func (cs *CredentialsStore) CheckReq(r *http.Request) bool {
	// read the authorization header of the HTTP request.
	username, password, ok := r.BasicAuth()
	return ok && cs.ValidatePassword(username, []byte(password))
}

// HasPermisson checks the store for a user's permissions and checks if that
// permission list contains the given permission.
func (cs *CredentialsStore) HasPermisson(username, permission string) bool {
	if mp, ok := cs.permissions[username]; ok {
		if _, ok := mp[permission]; ok {
			return true
		}
	}

	return false
}

// HasPermissionReq is a helper to directly check the HTTP request for the
// given permission using the more generic as.HasPermission() function.
func (cs *CredentialsStore) HasPermissionReq(r *http.Request, perm string) bool {
	username, _, ok := r.BasicAuth()
	if !ok {
		return false
	}

	return cs.HasPermisson(username, perm)
}
