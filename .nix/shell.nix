{ pkgs ? import ./pkgs.nix {} }:

let src = import ./src.nix;

in pkgs.mkShell {
	buildInputs = with pkgs; [
		gnome.gtk3
		glib
		gtk-layer-shell
		gdk-pixbuf
		gobjectIntrospection
		libhandy
	];

	nativeBuildInputs = with pkgs; [
		go
		pkgconfig
	];

	CGO_ENABLED = 1;

	# Use /tmp, since /run/user/1000 (XDG_RUNTIME_DIRECTORY) might be too small.
	# See https://github.com/NixOS/nix/issues/395.
	TMP    = "/tmp";
	TMPDIR = "/tmp";
}
