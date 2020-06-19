{ pkgs, buildGoModule, makeDesktopItem, lib }:

with import ./shell.nix { inherit pkgs; };

buildGoModule rec {
	inherit name;
	inherit version;
	inherit nativeBuildInputs;

	buildInputs = with pkgs; [
		gnome3.glib gnome3.gtk libhandy
	];

	src = ./.; # root Git directory

	desktopFile = makeDesktopItem {
		inherit name;
        desktopName = "gtkcord";
		exec = "gtkcord3";
		icon = "gtkcord3";
		categories = "GTK;GNOME;Chat;";
	};

	preFixup = ''
		mkdir -p $out/share/icons/hicolor/256x256/apps/ $out/share/applications/
		# Install the desktop file
		cp "${desktopFile}"/share/applications/* $out/share/applications/
		# Install the icon
		cp "${./logo.png}" $out/share/icons/hicolor/256x256/apps/gtkcord3.png
	'';

	vendorSha256 = "0jppx3m4qpp1k1k5b7xrw5pnlpbir7jmhvp2l06bgv2vpfzk0jqn";
	subPackages  = [ "." ];
}
