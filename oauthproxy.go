package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultBindAddress = ":53681"
)

var (
	configFile = flag.String("config", "oauthproxy-config.json", "location of config file")
	config     configData // global config
)

// configData describes the structure of the config file
type configData struct {
	AuthServer           string // outgoing auth server
	TokenServer          string // outoing token server
	ClientID             string // client ID to use with the auth/token server
	ClientSecret         string // client secret to use with the auth/token server
	IncomingClientID     string // client ID users must present to get access
	IncomingClientSecret string // client Secret users must present to get access
	BindAddress          string // address to bind to ":53681" by default
	Name                 string // name of service for title page
}

var htmlTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="en">
<head>
 <meta charset="utf-8">
 <title>{{ .Title }}</title>
</head>
<body>
 <h1>{{ .Title }}</h1>
 <p>This is an oauth proxy.</p>
 <p>Current server time is {{ .Time }}</p>
</body>
</html>
`))

type htmlParams struct {
	Title string
	Time  time.Time
}

// Serve an index page
func index(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	err := htmlTemplate.Execute(w, htmlParams{
		Title: "oauth2 proxy",
		Time:  time.Now(),
	})
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "template failed", http.StatusInternalServerError)
	}
}

// decodeAuthHeader turns a Basic Auth header into a user and pass
func decodeAuthHeader(authHeader string) (user, pass string, err error) {
	if !strings.HasPrefix(authHeader, "Basic ") {
		return "", "", errors.New("not Basic auth")
	}
	secret, err := base64.StdEncoding.DecodeString(authHeader[6:])
	if err != nil {
		return "", "", errors.New("base base64 encoding failed")
	}
	// check the username and password are correct
	parts := strings.SplitN(string(secret), ":", 2)
	if len(parts) != 2 {
		return "", "", errors.New("not user:pass")
	}
	return parts[0], parts[1], nil
}

// updateAuthHeader checks authHeader to make sure the credentials are
// correct and returns an updated version with new credentials
func updateAuthHeader(authHeader string) (newAuthHeader string, err error) {
	user, password, err := decodeAuthHeader(authHeader)
	if err != nil {
		return "", err
	}
	if user != config.IncomingClientID || password != config.IncomingClientSecret {
		return "", errors.New("bad username or password")
	}
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(config.ClientID+":"+config.ClientSecret)), nil
}

// This is a simple proxy which translates GET and POST requests
// replacing the following before redirecting
//  a URL parameter of client_id
//  an Authorized: header
//  the URL destination (different for /auth and /token)
func proxy(w http.ResponseWriter, req *http.Request) {
	// Select destination based on URL
	var serverURL string
	switch req.URL.Path {
	case "/auth":
		serverURL = config.AuthServer
	case "/token":
		serverURL = config.TokenServer
	default:
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Read up to 16K of the body
	body, err := ioutil.ReadAll(&io.LimitedReader{R: req.Body, N: 16384})
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}

	// Check and replace Authorization if present
	authHeader := req.Header.Get("Authorization")
	if authHeader != "" {
		authHeader, err = updateAuthHeader(authHeader)
		if err != nil {
			log.Printf("Authorization failed: %v", err)
			http.Error(w, "Authorization failed", http.StatusForbidden)
			return
		}
		// Replace authorization header with a the new one
		req.Header.Set("Authorization", authHeader)
	}

	// make outgoing URL
	outURL, err := url.Parse(serverURL)
	if err != nil {
		http.Error(w, "failed to parse URL", http.StatusInternalServerError)
		return
	}

	// include incoming URL parameters, replacing client_id if present
	query := req.URL.Query()
	if incomingClientID := query.Get("client_id"); incomingClientID != "" {
		if incomingClientID != config.IncomingClientID {
			log.Printf("Authorization failed: Bad Incoming Client ID")
			http.Error(w, "Authorization failed", http.StatusForbidden)
			return
		}
		query.Set("client_id", config.ClientID)
	}
	outURL.RawQuery = query.Encode()

	// outgoing request
	outReq, err := http.NewRequest(req.Method, outURL.String(), bytes.NewReader(body))
	if err != nil {
		http.Error(w, "failed to make NewRequest", http.StatusInternalServerError)
		return
	}

	// use (modified) headers from incoming request
	outReq.Header = req.Header

	// Do the HTTP round trip
	resp, err := http.DefaultClient.Do(outReq)
	if err != nil {
		log.Printf("fetch failed: %v", err)
		http.Error(w, "fetch failed", http.StatusInternalServerError)
		return
	}

	// copy the returned headers into the response
	header := w.Header()
	for k, vs := range resp.Header {
		for _, v := range vs {
			header.Add(k, v)
		}
	}

	// copy the response code
	w.WriteHeader(resp.StatusCode)

	// copy the returned body to the output
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Failed to write body: %v", err)
		http.Error(w, "failed to write body", http.StatusInternalServerError)
		return
	}
}

// loadConfigFile loads the config from the JSON file specified by -config
func loadConfigFile() {
	r, err := os.Open(*configFile)
	if err != nil {
		log.Fatalf("Failed to open config file %q: %v", *configFile, err)
	}
	defer r.Close()
	err = json.NewDecoder(r).Decode(&config)
	if err != nil {
		log.Fatalf("Failed to decode config file %q: %v", *configFile, err)
	}
}

// checkConfig makes sure that requred config is present
func checkConfig() (ok bool) {
	ok = true
	if config.AuthServer == "" {
		log.Printf("Config key AuthServer is required")
		ok = false
	}
	if config.TokenServer == "" {
		log.Printf("Config key TokenServer is required")
		ok = false
	}
	if config.ClientID == "" {
		log.Printf("Config key ClientID is required")
		ok = false
	}
	if config.ClientSecret == "" {
		log.Printf("Config key ClientSecret is required")
		ok = false
	}
	if config.IncomingClientID == "" {
		log.Printf("Config key IncomingClientID is required")
		ok = false
	}
	if config.IncomingClientSecret == "" {
		log.Printf("Config key IncomingClientSecret is required")
		ok = false
	}
	if config.BindAddress == "" {
		config.BindAddress = defaultBindAddress
	}
	if config.Name == "" {
		config.Name = "oauth proxy"
	}
	return
}

func main() {
	flag.Parse()
	loadConfigFile()
	if !checkConfig() {
		log.Fatalf("Missing data in config file %q", *configFile)
	}
	http.HandleFunc("/auth", proxy)
	http.HandleFunc("/token", proxy)
	http.HandleFunc("/", index)
	log.Fatal(http.ListenAndServe(config.BindAddress, nil))
}
