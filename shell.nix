{ pkgs ? import <nixpkgs> {} }:

pkgs.stdenv.mkDerivation rec {
	name = "gtkcord3";

	buildInputs = with pkgs; [
		gnome3.glib gnome3.gtk
		pkgconfig go gdb
	];
}
