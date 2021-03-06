package implicit

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/go-errors/errors"
	"github.com/golang/mock/gomock"
	"github.com/ory-am/fosite"
	"github.com/ory-am/fosite/fosite-example/store"
	"github.com/ory-am/fosite/handler/core/implicit"
	oauthStrat "github.com/ory-am/fosite/handler/core/strategy"
	"github.com/ory-am/fosite/handler/oidc"
	"github.com/ory-am/fosite/handler/oidc/strategy"
	"github.com/ory-am/fosite/internal"
	"github.com/ory-am/fosite/token/hmac"
	"github.com/ory-am/fosite/token/jwt"
	"github.com/stretchr/testify/assert"
)

var idStrategy = &strategy.DefaultStrategy{
	RS256JWTStrategy: &jwt.RS256JWTStrategy{
		PrivateKey: internal.MustRSAKey(),
	},
}

var hmacStrategy = &oauthStrat.HMACSHAStrategy{
	Enigma: &hmac.HMACStrategy{
		GlobalSecret: []byte("some-super-cool-secret-that-nobody-knows"),
	},
}

func TestHandleAuthorizeEndpointRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	aresp := fosite.NewAuthorizeResponse()
	areq := fosite.NewAuthorizeRequest()
	httpreq := &http.Request{Form: url.Values{}}

	h := OpenIDConnectImplicitHandler{
		AuthorizeImplicitGrantTypeHandler: &implicit.AuthorizeImplicitGrantTypeHandler{
			AccessTokenLifespan: time.Hour,
			AccessTokenStrategy: hmacStrategy,
			AccessTokenStorage:  store.NewStore(),
		},
		IDTokenHandleHelper: &oidc.IDTokenHandleHelper{
			IDTokenStrategy: idStrategy,
		},
	}
	for k, c := range []struct {
		description string
		setup       func()
		expectErr   error
		check       func()
	}{
		{
			description: "should not do anything because request requirements are not met",
			setup:       func() {},
		},
		{
			description: "should not do anything because request requirements are not met",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"id_token"}
			},
		},
		{
			description: "should not do anything because request requirements are not met",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"token", "id_token"}
			},
		},
		{
			description: "should not do anything because request requirements are not met",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{}
				areq.Scopes = fosite.Arguments{"openid"}
			},
		},
		{
			description: "should not do anything because request requirements are not met",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"token", "id_token"}
				areq.Scopes = fosite.Arguments{"openid"}
				areq.Client = &fosite.DefaultClient{
					GrantTypes:    fosite.Arguments{},
					ResponseTypes: fosite.Arguments{},
				}
			},
			expectErr: fosite.ErrInvalidGrant,
		},
		{
			description: "should not do anything because request requirements are not met",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"token", "id_token"}
				areq.Scopes = fosite.Arguments{"openid"}
				areq.Client = &fosite.DefaultClient{
					GrantTypes:    fosite.Arguments{"implicit"},
					ResponseTypes: fosite.Arguments{},
				}
			},
			expectErr: fosite.ErrInvalidGrant,
		},
		{
			description: "should fail because session not set",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"id_token"}
				areq.Scopes = fosite.Arguments{"openid"}
				areq.Client = &fosite.DefaultClient{
					GrantTypes:    fosite.Arguments{"implicit"},
					ResponseTypes: fosite.Arguments{"token", "id_token"},
				}
			},
			expectErr: oidc.ErrInvalidSession,
		},
		{
			description: "should fail because nonce not set",
			setup: func() {
				areq.Session = &strategy.DefaultSession{
					Claims: &jwt.IDTokenClaims{
						Subject: "peter",
					},
					Headers: &jwt.Headers{},
				}
				areq.Form.Add("nonce", "some-random-foo-nonce-wow")
			},
		},
		{
			description: "should pass",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"id_token"}
			},
			check: func() {
				assert.NotEmpty(t, aresp.GetFragment().Get("id_token"))
				assert.Empty(t, aresp.GetFragment().Get("access_token"))
			},
		},
		{
			description: "should pass",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"token", "id_token"}
			},
			check: func() {
				assert.NotEmpty(t, aresp.GetFragment().Get("id_token"))
				assert.NotEmpty(t, aresp.GetFragment().Get("access_token"))
			},
		},
		{
			description: "should pass",
			setup: func() {
				areq.ResponseTypes = fosite.Arguments{"id_token", "token"}
				areq.Scopes = fosite.Arguments{"fosite", "openid"}
			},
			check: func() {
				assert.NotEmpty(t, aresp.GetFragment().Get("id_token"))
				assert.NotEmpty(t, aresp.GetFragment().Get("access_token"))
			},
		},
	} {
		c.setup()
		err := h.HandleAuthorizeEndpointRequest(nil, httpreq, areq, aresp)
		assert.True(t, errors.Is(c.expectErr, err), "(%d) %s\n%s\n%s", k, c.description, err, c.expectErr)
		t.Logf("Passed test case %d", k)
		if c.check != nil {
			c.check()
		}
	}
}
