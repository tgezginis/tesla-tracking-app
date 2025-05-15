package tesla

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"crypto/tls"

	"github.com/go-resty/resty/v2"
)

const (
	ClientID              = "ownerapi"
	RedirectURI           = "https://auth.tesla.com/void/callback"
	AuthURL               = "https://auth.tesla.com/oauth2/v3/authorize"
	TokenURL              = "https://auth.tesla.com/oauth2/v3/token"
	Scope                 = "openid email offline_access"
	CodeChallengeMethod   = "S256"
	AppVersion            = "4.43.0-3212"
)

var (
	TokenFile  string
	OrdersFile string
)

func init() {
	configDir := getConfigDir()
	
	err := os.MkdirAll(configDir, 0700)
	if err != nil {
		fmt.Printf("Warning: Could not create config directory: %v\n", err)
	}
	
	TokenFile = filepath.Join(configDir, "tesla_tokens.json")
	OrdersFile = filepath.Join(configDir, "tesla_orders.json")
}

func getConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Tesla")
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Tesla")
	default:
		return filepath.Join(homeDir, ".config", "tesla")
	}
}

type TeslaAuth struct {
	CodeVerifier  string
	CodeChallenge string
	AccessToken   string
	RefreshToken  string
	Client        *resty.Client
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type Claim struct {
	Exp int64 `json:"exp"`
}

func NewTeslaAuth() *TeslaAuth {
	client := resty.New()
	
	client.SetTimeout(90 * time.Second)
	
	client.SetDoNotParseResponse(false)
	client.SetDisableWarn(true)
	client.SetTransport(&http.Transport{
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		DisableKeepAlives: false,
		MaxIdleConns: 100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout: 120 * time.Second,
	})
	
	client.SetRetryCount(3)
	client.SetRetryWaitTime(5 * time.Second)
	client.SetRetryMaxWaitTime(20 * time.Second)
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		return err != nil || r.StatusCode() >= 500
	})
	
	auth := &TeslaAuth{
		Client: client,
	}

	auth.GenerateCodeVerifierAndChallenge()

	return auth
}

func (a *TeslaAuth) GenerateCodeVerifierAndChallenge() {
	b := make([]byte, 32)
	rand.Read(b)
	a.CodeVerifier = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b)

	h := sha256.New()
	h.Write([]byte(a.CodeVerifier))
	a.CodeChallenge = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h.Sum(nil))
}

func (a *TeslaAuth) GetAuthURL() string {
	state := make([]byte, 16)
	rand.Read(state)
	stateStr := fmt.Sprintf("%x", state)
	
	params := url.Values{}
	params.Add("client_id", ClientID)
	params.Add("redirect_uri", RedirectURI)
	params.Add("response_type", "code")
	params.Add("scope", Scope)
	params.Add("state", stateStr)
	params.Add("code_challenge", a.CodeChallenge)
	params.Add("code_challenge_method", CodeChallengeMethod)
	
	return fmt.Sprintf("%s?%s", AuthURL, params.Encode())
}

func (a *TeslaAuth) ExchangeCodeForTokens(authCode string) error {
	formData := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     ClientID,
		"code":          authCode,
		"redirect_uri":  RedirectURI,
		"code_verifier": a.CodeVerifier,
	}
	
	resp, err := a.Client.R().
		EnableTrace().
		SetFormData(formData).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "TeslaGoClient/1.0").
		Post(TokenURL)
	
	if err != nil {
		return err
	}
	
	if resp.StatusCode() != 200 {
		return fmt.Errorf("failed to get token: %s", resp.String())
	}
	
	var tokenResp TokenResponse
	if err := json.Unmarshal(resp.Body(), &tokenResp); err != nil {
		return err
	}
	
	a.AccessToken = tokenResp.AccessToken
	a.RefreshToken = tokenResp.RefreshToken
	
	return nil
}

func (a *TeslaAuth) SaveTokensToFile() error {
	data := map[string]string{
		"access_token":  a.AccessToken,
		"refresh_token": a.RefreshToken,
	}
	
	fileData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	
	return os.WriteFile(TokenFile, fileData, 0600)
}

func (a *TeslaAuth) LoadTokensFromFile() error {
	fileData, err := os.ReadFile(TokenFile)
	if err != nil {
		return err
	}
	
	var data map[string]string
	if err := json.Unmarshal(fileData, &data); err != nil {
		return err
	}
	
	a.AccessToken = data["access_token"]
	a.RefreshToken = data["refresh_token"]
	
	return nil
}

func (a *TeslaAuth) IsTokenValid() bool {
	if a.AccessToken == "" {
		return false
	}
	
	parts := strings.Split(a.AccessToken, ".")
	if len(parts) != 3 {
		return false
	}
	
	payload := parts[1]
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", 4-len(payload)%4)
	}
	
	decodedBytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return false
	}
	
	var claims Claim
	if err := json.Unmarshal(decodedBytes, &claims); err != nil {
		return false
	}
	
	return claims.Exp > time.Now().Unix()
}

func (a *TeslaAuth) RefreshTokens() error {
	resp, err := a.Client.R().
		SetFormData(map[string]string{
			"grant_type":    "refresh_token",
			"client_id":     ClientID,
			"refresh_token": a.RefreshToken,
		}).
		Post(TokenURL)
	
	if err != nil {
		return err
	}
	
	if resp.StatusCode() != 200 {
		return fmt.Errorf("failed to refresh tokens: %s", resp.String())
	}
	
	var tokenResp TokenResponse
	if err := json.Unmarshal(resp.Body(), &tokenResp); err != nil {
		return err
	}
	
	a.AccessToken = tokenResp.AccessToken
	
	return nil
} 