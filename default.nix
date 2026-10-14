{
  buildGoModule,
  fetchFromGitHub,
  lib,
}:
buildGoModule (final: {
  pname = "kyma";
  version = "0.2.0";

  src = fetchFromGitHub {
    owner = "museslabs";
    repo = "kyma";
    rev = "v${final.version}";
    sha256 = "sha256-H9cgET4EjQj17Y3uWKCqN9FwcTA04P6vYFssV3MOuB4=";
  };

  vendorHash = "sha256-iDg/R7gnhoSQltXxC9Xg3WzWprdrHhQYD1giWL+EZvo=";

  meta = {
    description = "Presentations from markdown in the terminal with fancy transition animations";
    homepage = "https://github.com/museslabs/kyma";
    license = lib.licenses.gpl3;
    maintainers = with lib.maintainers; [ museslabs ];
  };

})
