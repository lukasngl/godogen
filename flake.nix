{
  description = "godogen - Go code generator for godog step definitions";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go

            # Build tools
            just
            git
            ripgrep
            gnumake

            # Formatters
            treefmt
            dprint
            shfmt
            shellcheck
            nodePackages.prettier

            # Go tools (using nixpkgs versions)
            golangci-lint
            gotools # includes goimports
            gofumpt
            gotestsum
            goreleaser
          ];

          shellHook = ''
            # Set up Go environment
            export GOPATH="$(pwd)/.go"
            export PATH="$GOPATH/bin:$PATH"

            echo "Development environment ready!"
            echo "Go version: $(go version)"
          '';
        };
      }
    );
}
