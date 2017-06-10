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

Now you need to configure rclone to use the new oauth proxy. You'll
need an up to date rclone 1.37 or an up to date beta of 1.36 to use
the oauth proxy.

Create a new remote with `rclone config`. When filling out the details
in the configurator note that:

  * `client_id` takes the value of `IncomingClientID`
  * `client_secret` takes the value of `IncomingClientSecret`
  * `auth_url` is the same as your appspot URL but with `/auth` on the end.
  * `token_url` is the same as your appspot URL but with `/token` on the end.

```
No remotes found - make a new one
n) New remote
s) Set configuration password
q) Quit config
n/s/q> n
name> acd
Type of storage to configure.
Choose a number from below, or type in your own value
 1 / Amazon Drive
   \ "amazon cloud drive"
 2 / Amazon S3 (also Dreamhost, Ceph, Minio)
   \ "s3"
 3 / Backblaze B2
   \ "b2"
 4 / Dropbox
   \ "dropbox"
 5 / Encrypt/Decrypt a remote
   \ "crypt"
 6 / FTP Connection
   \ "ftp"
 7 / Google Cloud Storage (this is not Google Drive)
   \ "google cloud storage"
 8 / Google Drive
   \ "drive"
 9 / Hubic
   \ "hubic"
10 / Local Disk
   \ "local"
11 / Microsoft OneDrive
   \ "onedrive"
12 / Openstack Swift (Rackspace Cloud Files, Memset Memstore, OVH)
   \ "swift"
13 / SSH/SFTP Connection
   \ "sftp"
14 / Yandex Disk
   \ "yandex"
Storage> 1
Amazon Application Client Id - required.
client_id> IncomingClientID
Amazon Application Client Secret - required.
client_secret> IncomingClientSecret
Auth server URL - leave blank to use Amazon's.
auth_url> https://YOURPROJECT.appspot.com/auth
Token server url - leave blank to use Amazon's.
token_url> https://YOURPROJECT.appspot.com/token
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
