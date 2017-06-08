# oauth2 proxy #

This is an oauth2 proxy designed to run on Google Appengine. Its
purpose is to allow users to connect to an oauth service while hiding
the client ID and client secret for the service from the users.

It works by receiving connections and proxying them onwards.  It
changes any client ID and client secrets as it passes through into the
secret version.

Authentication to this proxy is done with a different client ID and
client secret.  These must be correct for the proxy to use the secret
credentials.

It doesn't store any data or credentials as they pass through the
proxy.

It is designed to be compatible with any oauth2 application.

## Installation ##

Follow the [go appengine quickstart "before you begin" section](https://cloud.google.com/appengine/docs/standard/go/quickstart#before-you-begin).

You should now have a working `gcloud` tool at this point and a project
created with appengine before continuing.

Install the [go language](https://golang.org/doc/install).  Make sure
go is on your PATH by running `go version`.

Set GOPATH - this should be a specific one for this project.

    export GOPATH=/tmp/go-appengine

Install the appengine support libraries

    go get google.golang.org/appengine

Check out the code wherever you like

    git clone https://github.com/ncw/oauthproxy.git
    cd oauthproxy

## Configuration ##

You need to create a file like this called `oauthproxy-config.json`
(you can find this example in
[oauthproxy-config.json](/oauthproxy-config.json.example))

```json
{
    "AuthURL" : "https://www.amazon.com/ap/oa",
    "TokenURL" : "https://api.amazon.com/auth/o2/token",
    "ClientID" : "Your Client ID goes here",
    "ClientSecret" : "Your Client secret goes here",
    "IncomingClientID" : "Client ID (username) your users should use",
    "IncomingClientSecret" : "Client Secret (password) your users should use",
    "Name" : "oauth2 proxy"
}
```

Note that `ClientID` and `ClientSecret` should be the ones for the
service you are trying to use at `AuthURL` and `TokenURL`.

Whereas `IncomingClientID` and `IncomingClientSecret` should be made
up by you to be as secure or not as you like.  These are used for
authentication to this service in place of a username and password.

## Testing ##

Now try running the development version of this code.

    dev_appserver.py app.yaml

Check this starts up, and you can visit the test app at
http://localhost:8080/.  This should show a simple page with the
current time on.

## Deploy ##

You are ready to deploy the app to Appengine now.

    gcloud app deploy

Now visit the app to check it is working

    gcloud app browse

It will take you to the index page of the app.  Make a note of the URL
it will be something like `https://YOURPROJECT.appspot.com`.

## rclone configuration ##

First make sure your oauth credentials allow the redirect URL of
`http://127.0.0.1:53682/` as this is what rclone uses.  If you don't
set this then the authorization process will fail mysteriously.

Now you need to configure rclone to use the new oauth proxy.

You'll need to configure the config by hand and you'll rclone 1.37 or
a beta to use this.

Find the config like this and then edit it.

```
$ rclone -h | grep Config
      --config string                     Config file. (default "/home/USER/.rclone.conf")
```

The rclone config should look like this.  Note that `client_id` takes
the value of `IncomingClientID` and `client_secret` takes the value of
`IncomingClientSecret`.  Note that the URL for `auth_url` is the same
as your appspot URL but with `/auth` on the end.  Likewise the
`token_url` with `/token` on the end.

```
[acd]
type = amazon cloud drive
client_id = Client ID (username) your users should use
client_secret = Client ID (username) your users should use
auth_url = https://YOURPROJECT.appspot.com/auth
token_url = https://YOURPROJECT.appspot.com/token
```

Run through the config process to refresh the token

```
$ rclone config
e) Edit existing remote
n) New remote
d) Delete remote
r) Rename remote
c) Copy remote
s) Set configuration password
q) Quit config
e/n/d/r/c/s/q> e
Choose a number from below, or type in an existing value
 1 > acd
remote> acd
--------------------
[acd]
type = amazon cloud drive
client_id = IncomingClientID
client_secret = IncomingClientSecret
auth_url = https://YOURPROJECT.appspot.com/auth
token_url = https://YOURPROJECT.appspot.com/token
--------------------
Edit remote
Value "client_id" = "IncomingClientID"
Edit? (y/n)>
y) Yes
n) No
y/n> n
Value "client_secret" = "IncomingClientSecret"
Edit? (y/n)>
y) Yes
n) No
y/n> n
Remote config
Make sure your Redirect URL is set to "http://127.0.0.1:53682/" in your custom config.
Use auto config?
 * Say Y if not sure
 * Say N if you are working on a remote or headless machine
y) Yes
n) No
y/n> y
If your browser doesn't open automatically go to the following link: http://127.0.0.1:53682/auth
Log in and authorize rclone for access
Waiting for code...
Got code
--------------------
[acd]
type = amazon cloud drive
client_id = IncomingClientID
client_secret = IncomingClientSecret
auth_url = https://YOURPROJECT.appspot.com/auth
token_url = https://YOURPROJECT.appspot.com/token
token = XXXX
--------------------
y) Yes this is OK
e) Edit this remote
d) Delete this remote
y/e/d> y
```

rclone should be ready to use - test with `rclone lsd acd:`.
