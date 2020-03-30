{ pkgs ? import <nixpkgs> {} }:

pkgs.stdenv.mkDerivation rec {
	nix-bundle = pkgs.nix-bundle.overrideAttrs (_: {
		version = "0.3.1-pre";
		src = builtins.fetchGit {
			url = "https://github.com/matthewbauer/nix-bundle.git";
			rev = "4300437ede1f10c14cde157d9cce407bd46f5902";
		};
	});

	name = "gtkcord3";
	version = "0.0.2";

	buildInputs = with pkgs; [
		gnome3.glib gnome3.gtk libhandy
	];

	nativeBuildInputs = with pkgs; [
		pkgconfig go
	];

	shellHook = ''
		# gtkcord_appimage() {
		# 	out=$(nix-store --no-gc-warning -r $(nix-instantiate --no-gc-warning ./appimage.nix))
		# 	echo "$out"
		# }
	'';
}
