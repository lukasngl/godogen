# Changelog

## [0.4.0](https://github.com/lukasngl/godogen/compare/v0.3.1...v0.4.0) (2026-01-07)


### Features

* add CI/CD setup with Nix and GitHub Actions ([1fdac56](https://github.com/lukasngl/godogen/commit/1fdac56b2e8c445db0d5e3d84b559af1af7112c3))
* add GoReleaser for multi-platform binary releases ([ba2b717](https://github.com/lukasngl/godogen/commit/ba2b717566a16149266fad9c75552dae8c6eb848))
* add version management and Nix build packages ([386b11c](https://github.com/lukasngl/godogen/commit/386b11c76f767022e1b658297848caaea4a06085))
* **cli:** add CLI commands for workspace analysis ([ab52d49](https://github.com/lukasngl/godogen/commit/ab52d491ed2b369efbe21a884fcd99141e39929f))
* **language-server:** add feature specs for new diagnostics and LSP features ([5c69126](https://github.com/lukasngl/godogen/commit/5c691266ab51be3c19df232d54525090171cc1cd))
* **lsp:** add ambiguous step matches diagnostic ([3cda55f](https://github.com/lukasngl/godogen/commit/3cda55f4fec23ff72bd806b5f947433bd2c92247))
* **lsp:** add duplicate step definitions diagnostic ([ffd0c65](https://github.com/lukasngl/godogen/commit/ffd0c65e911b03b884ce56133d353834e9232593))
* **lsp:** add glob pattern support and config file loading ([5d7990e](https://github.com/lukasngl/godogen/commit/5d7990ec68c6235002673efc4d9df0fea643dcff))
* **lsp:** add Scenario Outline support with related info diagnostics ([5ab46f6](https://github.com/lukasngl/godogen/commit/5ab46f66cdb7cda83210055882858aefb277ddec))
* **lsp:** add textDocument/documentSymbol support ([a106f9c](https://github.com/lukasngl/godogen/commit/a106f9c453f2d55345d1293e94f0fd5ad0fb599b))
* **lsp:** add undefined steps diagnostic for feature files ([690c072](https://github.com/lukasngl/godogen/commit/690c072c6256f0b6f0c3c79de69d1121f5e89480))
* **lsp:** add unused step definitions diagnostic ([74b1d6c](https://github.com/lukasngl/godogen/commit/74b1d6c65c8c0e2f19be579a0041041d442e4f4c))
* **lsp:** complete gherkin keywords ([e8ac573](https://github.com/lukasngl/godogen/commit/e8ac573396cdc7da02bbfea855c8a80f3e5d7876))
* **lsp:** expose suggested fix as code action ([ddae876](https://github.com/lukasngl/godogen/commit/ddae87638aee22b228d855282133f966aa582fe0))
* **lsp:** expose validation errors as diagnostics ([21f1300](https://github.com/lukasngl/godogen/commit/21f13007422c411789064a03fabeaddd20bb0a49))
* **lsp:** find references from function declarations ([3ad32a5](https://github.com/lukasngl/godogen/commit/3ad32a5df9cee85889ae9ff22a11721a516f8f6a))
* **lsp:** goto definition for steps ([bcc09c5](https://github.com/lukasngl/godogen/commit/bcc09c50fb759f385bceba2c5027aabd0b72093f))
* **lsp:** goto implementation for steps ([3dc2b36](https://github.com/lukasngl/godogen/commit/3dc2b362776c9c2db2bb3af1af796b62b26e3966))
* **lsp:** goto referencess for step patterns ([2708146](https://github.com/lukasngl/godogen/commit/2708146bc7013770693386327313e13f380db603))
* **lsp:** implement textDocument/hover support ([a751e5b](https://github.com/lukasngl/godogen/commit/a751e5bbce8cbf714e1798a9831bd2ab728be3bd))
* **lsp:** very very bad file sync ([7e38552](https://github.com/lukasngl/godogen/commit/7e3855291213b2a16b8a009ff998a87bcc8a0066))


### Bug Fixes

* **ci:** use Nix shell with syft for GoReleaser ([f0bcac1](https://github.com/lukasngl/godogen/commit/f0bcac1e8bfcc8ad71db7a6607497e870d090119))
* **lsp:** fix duplicate detection, unused diagnostics, and hover formatting ([bf61f3f](https://github.com/lukasngl/godogen/commit/bf61f3fa00caf49b6af6bbb3a2fcd70d8cbd6d53))
* **lsp:** map diagnostic severity correctly in LSP response ([47069b5](https://github.com/lukasngl/godogen/commit/47069b51ecdd558c7d00cee36781ca201614b73c))
* **lsp:** restore find references support for function names ([28355ea](https://github.com/lukasngl/godogen/commit/28355ea7d95bfa4ccb414895efcb0af987b8922e))
* **lsp:** skip scenario outline placeholders in undefined step check ([e52fc13](https://github.com/lukasngl/godogen/commit/e52fc131d7f18802cdbaa49742283d9fe933425f))
* **lsp:** use push diagnostics instead of pull ([d84f372](https://github.com/lukasngl/godogen/commit/d84f372b8f8bfe8bc0d35a8697f4787cbd0d6dd4))
* **test:** fix step definitions for strict mode ([c5c06b6](https://github.com/lukasngl/godogen/commit/c5c06b6339fccfcce98863d127b5e49c105ae4b4))
