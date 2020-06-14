{ pkgs, pkgsStatic, buildGoModule, makeDesktopItem, lib }:

with import ./shell.nix { inherit pkgs; };

buildGoModule rec {
	inherit name;
	inherit version;
	inherit nativeBuildInputs;

	buildInputs = with pkgsStatic; [
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

	vendorSha256 = "0rjazx7a25d6vhmja5xrf30nzvk03g5pmk9i7fgiy2srn562gca4";
	subPackages  = [ "." ];
}
