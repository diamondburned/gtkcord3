{ pkgs ? import <nixpkgs> {} }:

pkgs.stdenv.mkDerivation rec {
	name = "gtkcord3";
	version = "0.0.2";

	buildInputs = with pkgs; [
		gnome3.glib gnome3.gtk libhandy
	];

	nativeBuildInputs = with pkgs; [
		pkgconfig go
	];
}
