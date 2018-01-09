package account

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/satori/go.uuid"
)

type AuthError int

const (
	ErrAuthFailed AuthError = iota
	ErrPermissionDenied
)

func (e AuthError) Error() string {
	switch e {
	case ErrAuthFailed:
		return "auth failed"
	}
	return "unknown"
}

type Perm int

const (
	PermGuest Perm = iota
	PermUser
	PermAdmin
)

type Account struct {
	UID      string
	Perm     Perm
	Password []byte
}

type AuthContext struct {
	Error   error
	Account *Account
}
type AuthHandlerOption struct {
	Perm       Perm
	ParseLogin bool
}
type AuthHandlerFunc func(http.ResponseWriter, *http.Request, *AuthContext)

type AuthAction struct {
	cookieName           string
	redisCli             redis.UniversalClient
	uFormName, pFormName string
	authExpires          time.Duration
	HmacKey              []byte
}

func (a *AuthAction) authUserPassword(username, password string) (*Account, string, error) {
	data, err := a.redisCli.HGet("auth:data", username).Bytes()
	if err != nil {
		return nil, "", err
	}
	acc := &Account{}
	err = json.Unmarshal(data, acc)
	if err != nil {
		return nil, "", err
	}
	h := hmac.New(sha256.New, a.HmacKey)
	h.Write([]byte(password))
	if hex.EncodeToString(h.Sum(nil)) == acc.Password {
		sessID := uuid.NewV4()
		return acc, sessID, nil
	}
	acc, sessID, authPassed = a.authUserPassword(username, password)
	if authPassed {
		err = a.redisCli.Set(a.sessionKey(sessID), username, a.authExpires).Err()
		if err != nil {
			authCtx.Error = err
		} else {
			http.SetCookie(w, &http.Cookie{
				Name:    a.cookieName,
				Value:   sessID,
				Expires: time.Now().Add(a.authExpires).UTC(),
			})
		}
	} else {
		authCtx.Error = ErrAuthFailed
	}
	return nil, "", false
}

func (a *AuthAction) authSessionID(sessID string) (*Account, bool) {

	return nil, false
}

func (a *AuthAction) sessionKey(sessID string) string {
	return fmt.Sprintf("auth:sessions:%s", sessID)
}

func (a *AuthAction) Handler(fn AuthHandlerFunc, opt *AuthHandlerOption) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			sessID     string
			acc        *Account
			authPassed bool
			authCtx    = &AuthContext{}
		)
		cookie, err := r.Cookie(a.cookieName)
		if err == nil {
			sessID = cookie.Value
		}
		if opt != nil && opt.ParseLogin {
			username := strings.TrimSpace(r.PostFormValue(a.uFormName))
			password := strings.TrimSpace(r.PostFormValue(a.pFormName))
		}
		if !authPassed && sessID != "" {
			acc, authPassed = a.authSessionID(sessID)
		}
		if !authPassed || (authPassed && opt != nil && opt.Perm > acc.Perm) {
			authCtx.Error = ErrPermissionDenied
		}
		fn(w, r, authCtx)
	}
}
