# gtkcord

A lightweight Discord client which uses GTK3 for the user interface.

![Screenshot of gtkcord](.readme-resources/images/screenshot1.png)

## It's time to ditch the Discord Electron application (soon).

- Lighter than the official Discord application
- Faster than the official Discord application
- Uses less system resources than the official Discord application
- Is just as easy to use as the official Discord application
- Uses your prefered GTK theme

## Build gtkcord
**Required:** `go` (1.13+), `gtk`


### 1. Set the TOKEN variable to your Discord key

(The developer of gtkcord is currently working on a login, so that you no longer need to set the TOKEN variable.)

You can find this key by:
- Press F12 in when Discord is open (to open the Inspector).
- Press F5 to refresh the page and go to the Network tab.
- Search `api library` and look for the "Authorization" header in the right column.
- Copy this token.

### 2. Compile and run GtkCord

```sh
export TOKEN="<your copied token here>"
go run .
```

## Current features

- [X] See a list with Discord servers
	- [X] Folders
	- [X] Async loading
- [X] See a list of channels
	- [X] Server banner
	- [X] Async loading
- [X] See the messages of the selected channel
	- [X] Emojis
	- [X] Async loading
	- [ ] Message reactions
	- [X] Rich content
		- [X] Images
		- [x] Embeds
- [X] Send messages
  - [X] Text
  - [X] Images & files
	- [ ] Emojis
	- [ ] Message reactions
- [x] Graphical login
	- [ ] Graphical logout
- [X] Hamburger menu
	- [X] User avatar view
	- [ ] Change the visibility of your online state
		- [ ] Custom Rich Presence
		- [ ] Rich Presence IPC server
	- [ ] About dialog

## Low priority

- [ ] Options menu with the same options which Discord has
- [ ] Voice chat support

## Known Bugs/Limitations

- [ ] Emojis always appear large
- [ ] Random crashes
- [ ] Thread (un)safety with Xorg/xcb
- [ ] Rampant concurrency
- [ ] Semaphore limits
