package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"os"
	"sync"
)

type UserStore struct {
	// map[username]data
	Users map[string]user `json:"users"`

	umut *sync.RWMutex
}

// NeedAuth returns whether authentication is required
func (u *UserStore) NeedAuth() bool {
	u.umut.RLock()
	defer u.umut.RUnlock()

	return len(u.Users) != 0
}

// IsValidUser returns whether the given username/credentials combination is valid
func (u *UserStore) IsValidUser(user, passwd string) bool {
	u.umut.RLock()
	defer u.umut.RUnlock()

	usr, ok := u.Users[user]
	if !ok {
		return false
	}

	return constantTimeEquals(usr.PasswordHash, hash(passwd))
}

// Save persists the current user data to disk
func (u *UserStore) Save() (err error) {
	filepath := getConfigPath(userFileName)

	u.umut.Lock()
	defer u.umut.Unlock()

	tmp := filepath + ".temp"
	f, err := os.Create(tmp)
	if err != nil {
		return
	}

	err = json.NewEncoder(f).Encode(u)
	if err != nil {
		f.Close()
		return
	}

	err = f.Close()
	if err != nil {
		return
	}

	return os.Rename(tmp, filepath)
}

// loadUsers loads all user data from disk
func loadUsers(filepath string) (u *UserStore, err error) {
	// in case of error we must return an empty UserStore, not nil
	u = &UserStore{
		Users: make(map[string]user),
		umut:  new(sync.RWMutex),
	}

	f, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(u)

	if u.Users == nil {
		u.Users = make(map[string]user)
	}

	return
}

type user struct {
	PasswordHash string `json:"password_hash"`
}

func hash(password string) string {
	h := sha256.New()

	_, err := h.Write([]byte(password))
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func constantTimeEquals(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
