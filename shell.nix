{
  pkgs ? import <nixpkgs> { },
}:
pkgs.mkShell {
  buildInputs = [
    pkgs.nodejs
    pkgs.nodePackages.typescript
    pkgs.esbuild
    pkgs.nodePackages.less
    pkgs.air
  ];

  shellHook = ''
    make watch
  '';
}
