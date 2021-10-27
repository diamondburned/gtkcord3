{ systemChannel ? <nixpkgs> }:

let systemPkgs = import systemChannel {
		overlays = [ (import ./overlay.nix) ];
	};
	lib = systemPkgs.lib;

	src  = import ./src.nix;
	pkgs = import src.nixpkgs {
		overlays = [ (import ./overlay.nix) ];
	};

in
	if (
		(systemPkgs.gnome or null != null) &&
		(lib.versionAtLeast systemPkgs.gnome.gtk3.version "3.27.24") &&
		(lib.versionAtLeast systemPkgs.go.version "1.17")
	)
	# Prefer the system's Nixpkgs if it's new enough.
	then systemPkgs
	# Else, fetch our own.
	else pkgs
