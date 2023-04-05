package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/usa4ev/urlshortner/internal/server/auth"
)

const CtxKeyUserID contextKey = 0 // key to a userID context value

type contextKey int

// AuthMW returns middleware that enriches the request context with UserID
func AuthMW(sessionMgr auth.SessionStoreLoader) func(next http.Handler) http.Handler{
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error
			var usrID string
	
			errHandler := func(err error) {
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}
			var token string
	
			cookie, err := r.Cookie("userID")
			if err != nil && !errors.Is(err, http.ErrNoCookie) {
				errHandler(err)
			} else if err == nil {
				token = cookie.Value
			}
	
			if token != "" {
				//token is set, look up the user
				usrID, err = auth.LoadUser(token, sessionMgr)
				if err != nil {
					errHandler(err)
				}
			} else {
				//token is not set, open new session
				usrID, token, err = auth.OpenSession(sessionMgr)
				if err != nil {
					errHandler(err)
				}
			}
	
			setCookie(w, "userID", token)
			next.ServeHTTP(w, ctxWithSession(r, usrID))
		})
	}
}  

func ctxWithSession(r *http.Request, usrID string) *http.Request {
	ctx := context.WithValue(r.Context(), CtxKeyUserID, usrID)
	return r.WithContext(ctx)
}

func setCookie(w http.ResponseWriter, name string, value string) {
	cookie := &http.Cookie{Name: name, Value: value}
	http.SetCookie(w, cookie)
}
