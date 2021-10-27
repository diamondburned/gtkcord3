let systemPkgs = import <nixpkgs> {};

in {
	gotk4 = systemPkgs.fetchFromGitHub {
		owner = "diamondburned";
		repo  = "gotk4";
		rev   = "4f507c20f8b07f4a87f0152fbefdc9a380042b83";
		hash  = "sha256:0zijivbyjfbb2vda05vpvq268i7vx9bhzlbzzsa4zfzzr9427w66";
	};
	gotk4-adw = systemPkgs.fetchFromGitHub {
		owner = "diamondburned";
		repo  = "gotk4-adwaita";
		rev   = "01f60b73109a41d6b28e09dce61c45486bdc401b";
		hash  = "sha256:1l57ygzg5az0pikn0skj0bwggbvfj21d36glkwpkyp7csxi8hzhr";
	};
	nixpkgs = systemPkgs.fetchFromGitHub {
		owner = "NixOS";
		repo  = "nixpkgs";
		rev   = "3fdd780";
		hash  = "sha256:0df9v2snlk9ag7jnmxiv31pzhd0rqx2h3kzpsxpj07xns8k8dghz";
	};
}
