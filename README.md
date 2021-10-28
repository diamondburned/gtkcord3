<p align="center">
	
<img width="128" src="logo.png" />
<h1 align="center">gtkcord</h1>
<p  align="center">A lightweight Discord client which uses GTK3 for the user interface.</p>

<img src=".readme-resources/images/screenshot7.png" />

</p>

gtkcord3 is back.

## Why???????

Because the official client is lagging too hard for me to ignore.

gtkcord3 **won't be receiving any new features**. It is being maintained at a
minimal level just to ensure that it's usable on my computer. Crashed will still
be fixed as I use the application, but that's about it.

## Build gtkcord3
**Required:** `go` (1.13+), `gtk`, `libhandy`, `pkgconfig` (refer to `shell.nix`)

```sh
go get github.com/diamondburned/gtkcord3 # auto updates
~/go/bin/gtkcord3 # $GOPATH/bin/gtkcord3 or $GOBIN/gtkcord3
```

## Logging in

![Login screen](.readme-resources/images/login.png)

1. Press F12 in when Discord is open (to open the Inspector).
2. Go to the Network tab then press F5 to refresh the page.
3. Search `api library` then look for the "Authorization" header in the right column.
5. Copy this token into the Token field, then click Login.
