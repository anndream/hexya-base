// Copyright 2017 NDP Systèmes. All Rights Reserved.
// See LICENSE file for full licensing details.

package controllers

import (
	"net/http"

	"github.com/npiganeau/yep/yep/models/security"
	"github.com/npiganeau/yep/yep/models/types"
	"github.com/npiganeau/yep/yep/server"
)

// LoginGet is called when the client calls the login page
func LoginGet(c *server.Context) {
	redirect := c.DefaultQuery("redirect", "/web")
	if c.Session().Get("uid") != nil {
		c.Redirect(http.StatusSeeOther, redirect)
		return
	}
	data := struct{ ErrorMsg string }{ErrorMsg: ""}
	c.HTML(http.StatusOK, "web.login", data)
}

// LoginPost is called when the client sends credentials
// from the login page
func LoginPost(c *server.Context) {
	login := c.DefaultPostForm("login", "")
	secret := c.DefaultPostForm("password", "")
	uid, err := security.AuthenticationRegistry.Authenticate(login, secret, new(types.Context))
	if err != nil {
		data := struct{ ErrorMsg string }{ErrorMsg: "Wrong login or password"}
		c.HTML(http.StatusOK, "web.login", data)
		return
	}

	sess := c.Session()
	sess.Set("uid", uid)
	sess.Set("login", login)
	sess.Save()
	redirect := c.DefaultPostForm("redirect", "/web")
	c.Redirect(http.StatusSeeOther, redirect)
}

// LoginRequired is a middleware that redirects to login page
// non logged in users.
func LoginRequired(c *server.Context) {
	if c.Session().Get("uid") == nil {
		c.Redirect(http.StatusSeeOther, "/web/login")
		c.Abort()
	}
}
