{ pkgs ? import <nixpkgs> {} }:

with import ./shell.nix { inherit pkgs; };

let pkg = pkgs.callPackage ./default.nix {};

in (pkg // {
	bundle = nix-bundle.nix-bootstrap {
		run    = "/bin/gtkcord3";
		target = pkg;
		nixUserChrootFlags = "-p DISPLAY -p GTK_THEME";
	};
})
