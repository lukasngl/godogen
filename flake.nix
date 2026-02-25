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
        lib = pkgs.lib;

        # Source filtering for godogen (root package)
        godogenSrc = lib.fileset.toSource {
          root = ./.;
          fileset = lib.fileset.unions [
            ./go.mod
            ./main.go
            ./pkg
            ./version.txt
          ];
        };

        # Source filtering for godogen-lint (includes shared deps for replace directive)
        godogenLintSrc = lib.fileset.toSource {
          root = ./.;
          fileset = lib.fileset.unions [
            ./godogen-lint
            ./go.mod
            ./pkg
          ];
        };

        # Source filtering for godogen-language-server (includes shared deps for replace directive)
        godogenLanguageServerSrc = lib.fileset.toSource {
          root = ./.;
          fileset = lib.fileset.unions [
            ./godogen-language-server
            ./go.mod
            ./pkg
          ];
        };
      in
      {
        packages = {
          godogen = pkgs.buildGoModule {
            pname = "godogen";
            version = builtins.readFile ./version.txt;
            src = godogenSrc;
            vendorHash = null;
            subPackages = [ "." ];
            ldflags = [
              "-s"
              "-w"
            ];
            env.CGO_ENABLED = "0";
            meta = {
              description = "Go code generator for godog step definitions";
              homepage = "https://github.com/lukasngl/godogen";
              license = pkgs.lib.licenses.mit;
              mainProgram = "godogen";
            };
          };

          godogen-language-server = pkgs.buildGoModule {
            pname = "godogen-language-server";
            version = builtins.readFile ./godogen-language-server/version.txt;
            src = godogenLanguageServerSrc;
            modRoot = "godogen-language-server";
            vendorHash = "sha256-BwfqglO61HUY8WqaEJpnpKZ4kQ7dhIzC0I0OQRcYvsg=";
            ldflags = [
              "-s"
              "-w"
            ];
            env.CGO_ENABLED = "0";
            meta = {
              description = "Language server for godogen";
              homepage = "https://github.com/lukasngl/godogen";
              license = pkgs.lib.licenses.mit;
              mainProgram = "godogen-language-server";
            };
          };

          godogen-lint = pkgs.buildGoModule {
            pname = "godogen-lint";
            version = builtins.readFile ./godogen-lint/version.txt;
            src = godogenLintSrc;
            modRoot = "godogen-lint";
            vendorHash = "sha256-XnSyGjfFSurCoYs8o1xCrSwEpEf9dAyl9Y8OKjL4ybM=";
            ldflags = [
              "-s"
              "-w"
            ];
            env.CGO_ENABLED = "0";
            meta = {
              description = "Linter for godogen step definitions";
              homepage = "https://github.com/lukasngl/godogen";
              license = pkgs.lib.licenses.mit;
              mainProgram = "godogen-lint";
            };
          };

          default = pkgs.symlinkJoin {
            name = "godogen-tools";
            version = builtins.readFile ./version.txt;
            paths = [
              self.packages.${system}.godogen
              self.packages.${system}.godogen-lint
              self.packages.${system}.godogen-language-server
            ];
          };
        };

        devShells.tools = pkgs.mkShell {
          packages = [
            self.packages.${system}.godogen
            self.packages.${system}.godogen-lint
            self.packages.${system}.godogen-language-server
          ];
        };

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
            nixfmt-rfc-style
            keep-sorted

            # Go tools (using nixpkgs versions)
            golangci-lint
            gotools # includes goimports
            (gci.override { buildGoModule = pkgs.buildGo124Module; })
            gofumpt
            gotestsum
            goreleaser
            syft
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
