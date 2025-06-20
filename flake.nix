{
  description = "Go dev flake";

  inputs = {nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";};

  outputs = {
    self,
    nixpkgs,
  }: let
    allSystems = ["x86_64-linux" "aarch64-darwin"];
    forAllSystems = fn:
      nixpkgs.lib.genAttrs allSystems
      (system: fn {pkgs = import nixpkgs {inherit system;};});
  in {
    devShells = forAllSystems ({pkgs}: {
      default = pkgs.mkShell {
        nativeBuildInputs = with pkgs; [
            go
            gopls
            openapi-generator-cli
        ];
      };
    });
  };
}
