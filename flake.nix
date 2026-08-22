{
  description = "proto2mcp — Generate type-safe MCP servers from Protobuf";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go_1_25               # Go 1.25 (matches go.mod)
            buf                   # Proto linting & codegen
            golangci-lint         # Go linter
            lefthook              # Git hooks manager
            editorconfig-checker  # .editorconfig enforcement
            yamllint              # YAML linting
            goreleaser            # Release automation
          ];

          shellHook = ''
            if ! lefthook install > /dev/null 2>&1; then
              echo "⚠️  lefthook install failed — git hooks are NOT active."
              echo ""
            fi
            echo "proto2mcp dev shell — go $(go version | awk '{print $3}'), buf $(buf --version 2>&1)"
          '';
        };
      });
}
