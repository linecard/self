{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachSystem [
      "x86_64-linux"
      "aarch64-linux"
      "aarch64-darwin"
    ] (system:
      let
        pkgs = import nixpkgs { inherit system; };
        version = builtins.substring 0 8 self.lastModifiedDate;
      in
      rec {
        packages = rec {
          default = pkgs.buildGo122Module rec {
            pname = "self";
            inherit version;
            src = ./.;
            subPackages = [ "cmd/self" ];
            vendorHash = "sha256-E0TSkI5jVQsmU8FBwnWQfn0wsmM87mPjwuSBa4Do6BE=";
          };
        };

        devShell = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            go-tools
            gotools
            gopls
            ruby
            bundler
          ];

          AWS_REGION = "us-west-2";
          AWS_PROFILE = "linecard";
        };
      });
}
