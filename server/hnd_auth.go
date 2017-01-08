package server

import "net/http"

func loginHandler(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	v := struct {
		User     string
		Password string
	}{}

	if ok, code, msg := decodeBody(rw, r, &v); !ok {
		return code, msg, nil
	}

	token := login(v.User, v.Password)
	if token == "" {
		return rUnauth, "username or password were wrong", nil
	}

	setCookie(rw, token)

	// authenticated if we got here
	return rOK, "", struct {
		User  string
		Role  string
		Token string
	}{v.User, "ADMIN", token}
}

func logoutHandler(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	if ctx.sesh == nil {
		return rBad, "token not supplied", nil
	}
	logout(ctx.sesh.token)
	return rOK, "logged out", nil
}
